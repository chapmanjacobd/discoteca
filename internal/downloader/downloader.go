// Package downloader runs lb or lb-wrapper downloads and imports the results.
package downloader

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/logging"
	"github.com/chapmanjacobd/discoteca/internal/media"
	"github.com/chapmanjacobd/discoteca/internal/metadata"
)

// State values reported by a download job.
const (
	StateIdle      = "idle"
	StateRunning   = "running"
	StateCompleted = "completed"
	StateFailed    = "failed"
	StateCanceled  = "canceled"
)

// ErrBusy is returned when a download is already running. The manager exposes
// only one active job so downloads cannot saturate the host unexpectedly.
var ErrBusy = errors.New("a download is already running")

const maxOutputLines = 200

var (
	progressRE = regexp.MustCompile(`(?i)(\d{1,3}(?:\.\d+)?)%`)
	ansiRE     = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
)

// Kind identifies how the configured command expects its arguments.
type Kind string

const (
	KindLB      Kind = "lb"
	KindWrapper Kind = "lb-wrapper"
)

// Config is the resolved downloader command and output directory.
type Config struct {
	Command     string
	Kind        Kind
	DownloadDir string
}

// Options are the per-download settings supplied by the web client.
type Options struct {
	URL                string `json:"url"`
	Database           string `json:"database"`
	MediaType          string `json:"media_type"`
	Subtitles          string `json:"subtitles"`
	AudioFormat        string `json:"audio_format"`
	CookiesFromBrowser string `json:"cookies_from_browser"`
	CookiesFile        string `json:"cookies_file"`
	DownloadArchive    string `json:"download_archive"`
	ExtraMetadata      bool   `json:"extra_metadata"`
	Safe               bool   `json:"safe"`
}

// Normalize validates and canonicalizes options from the API.
func (o *Options) Normalize() error {
	o.URL = strings.TrimSpace(o.URL)
	parsed, err := url.Parse(o.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return errors.New("download URL must start with http:// or https://")
	}

	switch o.MediaType {
	case "", "auto":
		o.MediaType = "auto"
	case "video", "audio":
	default:
		return errors.New("media type must be auto, video, or audio")
	}

	switch o.Subtitles {
	case "", "none":
		o.Subtitles = "none"
	case "download", "auto":
	default:
		return errors.New("subtitles must be none, download, or auto")
	}

	o.Database = strings.TrimSpace(o.Database)
	o.CookiesFromBrowser = strings.TrimSpace(o.CookiesFromBrowser)
	o.CookiesFile = strings.TrimSpace(o.CookiesFile)
	o.DownloadArchive = strings.TrimSpace(o.DownloadArchive)

	if o.MediaType != "audio" {
		o.AudioFormat = ""
		return nil
	}
	switch o.AudioFormat {
	case "":
		o.AudioFormat = "opus"
	case "opus", "mp3", "m4a", "flac", "wav", "ogg":
	default:
		return errors.New("unsupported audio format")
	}
	return nil
}

// Status is a point-in-time snapshot of a download job.
type Status struct {
	ID           string   `json:"id"`
	URL          string   `json:"url"`
	State        string   `json:"state"`
	Progress     float64  `json:"progress"`
	ProgressText string   `json:"progress_text"`
	Output       []string `json:"output"`
	Error        string   `json:"error"`
	Imported     int      `json:"imported"`
	Paths        []string `json:"paths"`
	StartedAt    int64    `json:"started_at"`
	EndedAt      int64    `json:"ended_at"`
}

func (s Status) clone() Status {
	clone := s
	clone.Output = make([]string, len(s.Output))
	copy(clone.Output, s.Output)
	clone.Paths = make([]string, len(s.Paths))
	copy(clone.Paths, s.Paths)
	return clone
}

type job struct {
	status Status
	cancel context.CancelFunc
	done   chan struct{}
}

// Manager owns at most one active download job.
type Manager struct {
	cfg Config

	mu      sync.Mutex
	current *job
	closed  bool
}

// Resolve locates the downloader command and prepares the output directory.
func Resolve(command, downloadDir string) (Config, error) {
	if strings.TrimSpace(downloadDir) == "" {
		return Config{}, errors.New("download directory is empty")
	}
	absDir, err := filepath.Abs(downloadDir)
	if err != nil {
		return Config{}, fmt.Errorf("failed to resolve download directory: %w", err)
	}
	if mkErr := os.MkdirAll(absDir, 0o755); mkErr != nil {
		return Config{}, fmt.Errorf("failed to create download directory: %w", mkErr)
	}

	command = strings.TrimSpace(command)
	if command == "" || command == "auto" {
		if path, lookErr := exec.LookPath("lb"); lookErr == nil {
			return Config{Command: path, Kind: KindLB, DownloadDir: absDir}, nil
		}
		if path, lookErr := exec.LookPath("lb-wrapper"); lookErr == nil {
			return Config{Command: path, Kind: KindWrapper, DownloadDir: absDir}, nil
		}
		return Config{}, errors.New("neither lb nor lb-wrapper was found in PATH")
	}

	path, err := exec.LookPath(command)
	if err != nil {
		return Config{}, fmt.Errorf("downloader command %q not found: %w", command, err)
	}
	kind := KindLB
	if strings.Contains(strings.ToLower(filepath.Base(path)), "wrapper") {
		kind = KindWrapper
	}
	return Config{Command: path, Kind: kind, DownloadDir: absDir}, nil
}

// NewManager resolves the downloader and creates a job manager.
func NewManager(command, downloadDir string) (*Manager, error) {
	cfg, err := Resolve(command, downloadDir)
	if err != nil {
		return nil, err
	}
	return &Manager{cfg: cfg}, nil
}

// Config returns the resolved downloader configuration.
func (m *Manager) Config() Config {
	return m.cfg
}

// Start begins a download. Only one job may run at a time.
func (m *Manager) Start(opts Options) (*Status, error) {
	if err := opts.Normalize(); err != nil {
		return nil, err
	}
	if m.cfg.Kind == KindWrapper && opts.MediaType == "audio" {
		return nil, errors.New("lb-wrapper supports video downloads only; use lb for audio")
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, errors.New("downloader is closed")
	}
	if m.current != nil && m.current.status.State == StateRunning {
		m.mu.Unlock()
		return nil, ErrBusy
	}
	jobCtx, cancel := context.WithCancel(context.Background())
	j := &job{
		status: Status{
			ID:        strconv.FormatInt(time.Now().UnixNano(), 36),
			URL:       opts.URL,
			State:     StateRunning,
			Progress:  -1,
			Output:    []string{},
			Paths:     []string{},
			StartedAt: time.Now().Unix(),
		},
		cancel: cancel,
		done:   make(chan struct{}),
	}
	m.current = j
	initial := j.status.clone()
	m.mu.Unlock()

	go m.run(jobCtx, j, opts)
	return &initial, nil
}

// Status returns a snapshot of the current or most recent job, or nil.
func (m *Manager) Status() *Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return nil
	}
	status := m.current.status.clone()
	return &status
}

// Cancel stops the active job.
func (m *Manager) Cancel() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.status.State != StateRunning {
		return false
	}
	m.current.cancel()
	return true
}

// Close stops the active job and releases the manager.
func (m *Manager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	j := m.current
	if j != nil {
		j.cancel()
	}
	m.mu.Unlock()

	if j != nil {
		select {
		case <-j.done:
		case <-time.After(5 * time.Second):
		}
	}
}

func (m *Manager) run(ctx context.Context, j *job, opts Options) {
	defer close(j.done)
	defer j.cancel()

	before := snapshotFiles(m.cfg.DownloadDir)
	lw := &lineWriter{onLine: func(line string) { m.appendOutput(j, line) }}

	if m.cfg.Kind == KindWrapper {
		m.appendOutput(j, "lb-wrapper mode uses its own video defaults and output template")
		if err := m.runCommand(ctx, j, lw, commandSpec{
			label: "tubeadd",
			args:  []string{"tubeadd", opts.URL},
			env:   wrapperEnv(m.cfg.DownloadDir),
		}); err != nil {
			m.failOrCancel(ctx, j, err)
			return
		}
		if err := m.runCommand(ctx, j, lw, commandSpec{
			label: "download",
			args:  []string{"dl", opts.URL},
			env:   wrapperEnv(m.cfg.DownloadDir),
		}); err != nil {
			m.failOrCancel(ctx, j, err)
			return
		}
	} else {
		if err := m.runCommand(ctx, j, lw, commandSpec{
			label: "download",
			args:  lbArgs(m.cfg.DownloadDir, opts),
		}); err != nil {
			m.failOrCancel(ctx, j, err)
			return
		}
	}
	lw.Flush()

	if ctx.Err() != nil {
		m.finish(j, StateCanceled, "download canceled", nil, 0)
		return
	}

	newFiles := newMediaFiles(before, snapshotFiles(m.cfg.DownloadDir))
	if len(newFiles) == 0 {
		m.finish(j, StateCompleted, "", nil, 0)
		return
	}

	imported, err := importFiles(ctx, opts.Database, newFiles)
	if err != nil {
		m.finish(j, StateFailed, "download completed but import failed: "+err.Error(), newFiles, imported)
		return
	}
	m.finish(j, StateCompleted, "", newFiles, imported)
}

type commandSpec struct {
	label string
	args  []string
	env   []string
}

func (m *Manager) runCommand(ctx context.Context, j *job, lw *lineWriter, spec commandSpec) error {
	commandLine := append([]string{filepath.Base(m.cfg.Command)}, spec.args...)
	m.appendOutput(j, spec.label+": "+strings.Join(commandLine, " "))

	cmd := exec.CommandContext(ctx, m.cfg.Command, spec.args...)
	if len(spec.env) > 0 {
		cmd.Env = append(os.Environ(), spec.env...)
	}
	cmd.Stdout = lw
	cmd.Stderr = lw

	err := cmd.Run()
	lw.Flush()
	return err
}

func (m *Manager) failOrCancel(ctx context.Context, j *job, err error) {
	if ctx.Err() != nil {
		m.finish(j, StateCanceled, "download canceled", nil, 0)
		return
	}
	m.finish(j, StateFailed, err.Error(), nil, 0)
}

func (m *Manager) appendOutput(j *job, line string) {
	line = ansiRE.ReplaceAllString(strings.TrimSpace(line), "")
	if line == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if len(j.status.Output) >= maxOutputLines {
		copy(j.status.Output, j.status.Output[1:])
		j.status.Output[len(j.status.Output)-1] = line
	} else {
		j.status.Output = append(j.status.Output, line)
	}
	j.status.ProgressText = line
	if match := progressRE.FindStringSubmatch(line); len(match) > 1 {
		if pct, err := strconv.ParseFloat(match[1], 64); err == nil {
			j.status.Progress = min(pct, 100)
		}
	}
}

func (m *Manager) finish(j *job, state, errMsg string, paths []string, imported int) {
	m.mu.Lock()
	j.status.State = state
	j.status.EndedAt = time.Now().Unix()
	j.status.Error = errMsg
	j.status.Imported = imported
	j.status.Paths = make([]string, len(paths))
	copy(j.status.Paths, paths)
	if state == StateCompleted {
		j.status.Progress = 100
		j.status.ProgressText = "completed"
	}
	status := j.status.clone()
	m.mu.Unlock()

	logging.Log.Info("Download finished",
		"url", status.URL,
		"state", status.State,
		"imported", status.Imported,
		"error", status.Error,
	)
}

func lbArgs(downloadDir string, opts Options) []string {
	args := []string{
		"dl", opts.URL,
		"--prefix", downloadDir,
		"--media-type", opts.MediaType,
	}

	switch opts.Subtitles {
	case "download":
		args = append(args, "--subs")
	case "auto":
		args = append(args, "--auto-subs")
	}
	if opts.MediaType == "audio" && opts.AudioFormat != "" {
		args = append(args, "--extract-audio-ext", opts.AudioFormat)
	}
	if opts.CookiesFromBrowser != "" {
		args = append(args, "--cookies-from-browser", opts.CookiesFromBrowser)
	}
	if opts.CookiesFile != "" {
		args = append(args, "--cookies", opts.CookiesFile)
	}
	if opts.DownloadArchive != "" {
		args = append(args, "--download-archive", opts.DownloadArchive)
	}
	if opts.ExtraMetadata {
		args = append(args, "--extra")
	}
	if opts.Safe {
		args = append(args, "--safe")
	}
	return args
}

func wrapperEnv(downloadDir string) []string {
	return []string{
		"TMP_DOWNLOADS_DIR=" + downloadDir,
		"XKLB_DB_FILE=" + filepath.Join(downloadDir, "xklb-metadata.db"),
		"XKLB_LOG_FILE=" + filepath.Join(downloadDir, "lb-wrapper.log"),
	}
}

type lineWriter struct {
	mu     sync.Mutex
	buf    []byte
	onLine func(string)
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf = append(w.buf, p...)
	for {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:idx]), "\r")
		w.buf = w.buf[idx+1:]
		if w.onLine != nil {
			w.onLine(line)
		}
	}
	return len(p), nil
}

func (w *lineWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buf) == 0 {
		return
	}
	line := strings.TrimRight(string(w.buf), "\r\n")
	w.buf = nil
	if w.onLine != nil {
		w.onLine(line)
	}
}

func snapshotFiles(root string) map[string]struct{} {
	files := make(map[string]struct{})
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable entries instead of aborting the snapshot
		}
		if !d.IsDir() {
			files[path] = struct{}{}
		}
		return nil
	})
	return files
}

func newMediaFiles(before, after map[string]struct{}) []string {
	changed := make([]string, 0, len(after))
	for path := range after {
		if _, exists := before[path]; !exists {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)

	mediaStems := make(map[string]bool)
	for _, path := range changed {
		ext := strings.ToLower(filepath.Ext(path))
		if media.MediaExtensionMap[ext] && !media.ImageExtensionMap[ext] && !media.IsSubtitlePath(path) {
			mediaStems[strings.TrimSuffix(path, ext)] = true
		}
	}

	result := make([]string, 0, len(changed))
	for _, path := range changed {
		ext := strings.ToLower(filepath.Ext(path))
		if !media.MediaExtensionMap[ext] || media.IsSubtitlePath(path) {
			continue
		}
		if media.ImageExtensionMap[ext] {
			stem := strings.TrimSuffix(path, ext)
			base := strings.ToLower(filepath.Base(path))
			if mediaStems[stem] || isAuxiliaryImage(base) {
				continue
			}
		}
		result = append(result, path)
	}
	return result
}

func isAuxiliaryImage(base string) bool {
	for _, prefix := range []string{"cover.", "folder.", "poster.", "thumb."} {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}
	return false
}

func importFiles(ctx context.Context, dbPath string, paths []string) (int, error) {
	if strings.TrimSpace(dbPath) == "" {
		return 0, errors.New("no target database was selected")
	}

	sqlDB, queries, err := db.ConnectWithInit(ctx, dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open target database: %w", err)
	}
	defer sqlDB.Close()

	mediaBatch := make([]db.UpsertMediaParams, 0, len(paths))
	captionBatch := make([]db.InsertCaptionParams, 0, len(paths))
	for _, path := range paths {
		extracted, extractErr := metadata.Extract(ctx, path, metadata.ExtractOptions{ScanSubtitles: true})
		if extractErr != nil {
			logging.Log.Warn("Failed to import downloaded media", "path", path, "error", extractErr)
			continue
		}
		mediaBatch = append(mediaBatch, extracted.Media)
		captionBatch = append(captionBatch, extracted.Captions...)
	}
	if len(mediaBatch) == 0 {
		return 0, nil
	}

	if err := queries.BulkUpsertMedia(ctx, mediaBatch); err != nil {
		return 0, fmt.Errorf("failed to upsert downloaded media: %w", err)
	}
	if err := queries.BulkInsertCaptions(ctx, captionBatch); err != nil {
		return 0, fmt.Errorf("failed to insert downloaded captions: %w", err)
	}

	if err := db.RebuildFTS(ctx, sqlDB, dbPath); err != nil {
		logging.Log.Warn("Failed to rebuild FTS after download", "error", err)
	}
	if err := db.RefreshFolderStats(ctx, sqlDB); err != nil {
		logging.Log.Warn("Failed to refresh folder_stats after download", "error", err)
	}
	return len(mediaBatch), nil
}
