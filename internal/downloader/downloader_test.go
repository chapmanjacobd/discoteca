package downloader_test

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chapmanjacobd/discoteca/internal/downloader"
)

func writeFakeDownloader(t *testing.T, dir, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake downloader needs a POSIX shell")
	}

	script := `#!/bin/sh
set -e
cmd="$1"
shift
dir=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "--prefix" ]; then
    dir="$arg"
  fi
  prev="$arg"
done
if [ -z "$dir" ]; then
  dir="${TMP_DOWNLOADS_DIR:-}"
fi
if [ -z "$dir" ]; then
  echo "no download directory" >&2
  exit 1
fi
if [ "$cmd" = "tubeadd" ]; then
  mkdir -p "$dir"
  exit 0
fi
mkdir -p "$dir/Fake/Channel"
printf 'fake video' > "$dir/Fake/Channel/video.mp4"
printf 'fake thumbnail' > "$dir/Fake/Channel/video.webp"
cat > "$dir/Fake/Channel/video.en.vtt" <<'VTT'
WEBVTT

00:00:15.000 --> 00:00:16.000
downloaded caption
VTT
`
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func waitForJob(t *testing.T, manager *downloader.Manager) downloader.Status {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		status := manager.Status()
		if status != nil && status.State != downloader.StateRunning {
			return *status
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("download job did not finish")
	return downloader.Status{}
}

func TestDownloadAndImport(t *testing.T) {
	commandDir := t.TempDir()
	downloadDir := t.TempDir()
	command := writeFakeDownloader(t, commandDir, "lb")
	dbPath := filepath.Join(t.TempDir(), "media.db")

	manager, err := downloader.NewManager(command, downloadDir)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	status, err := manager.Start(downloader.Options{
		URL:       "https://example.test/video",
		Database:  dbPath,
		MediaType: "auto",
		Subtitles: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if status.State != downloader.StateRunning {
		t.Fatalf("initial state = %q", status.State)
	}

	final := waitForJob(t, manager)
	if final.State != downloader.StateCompleted {
		t.Fatalf("state = %q, error = %q, output = %v", final.State, final.Error, final.Output)
	}
	if final.Imported != 1 {
		t.Errorf("imported = %d, want 1", final.Imported)
	}

	if len(final.Paths) != 1 || filepath.Base(final.Paths[0]) != "video.mp4" {
		t.Errorf("paths = %v", final.Paths)
	}

	sqlDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	var path, mediaType string
	if err := sqlDB.QueryRow(`SELECT path, media_type FROM media`).Scan(&path, &mediaType); err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(downloadDir, "Fake", "Channel", "video.mp4")
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}
	if mediaType != "video" {
		t.Errorf("media_type = %q, want video", mediaType)
	}

	var mediaCount, captionCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM media`).Scan(&mediaCount); err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM captions WHERE text LIKE '%downloaded caption%'`).
		Scan(&captionCount); err != nil {
		t.Fatal(err)
	}
	if mediaCount != 1 {
		t.Errorf("media rows = %d, want 1 (thumbnail must not be imported)", mediaCount)
	}
	if captionCount != 1 {
		t.Errorf("captions = %d, want 1", captionCount)
	}
}

func TestWrapperDownload(t *testing.T) {
	commandDir := t.TempDir()
	downloadDir := t.TempDir()
	command := writeFakeDownloader(t, commandDir, "lb-wrapper")
	dbPath := filepath.Join(t.TempDir(), "media.db")

	manager, err := downloader.NewManager(command, downloadDir)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	if manager.Config().Kind != downloader.KindWrapper {
		t.Fatalf("kind = %q, want lb-wrapper", manager.Config().Kind)
	}

	if _, err := manager.Start(downloader.Options{
		URL:      "https://example.test/video",
		Database: dbPath,
	}); err != nil {
		t.Fatal(err)
	}

	final := waitForJob(t, manager)
	if final.State != downloader.StateCompleted {
		t.Fatalf("state = %q, error = %q, output = %v", final.State, final.Error, final.Output)
	}
	if final.Imported != 1 {
		t.Errorf("imported = %d, want 1", final.Imported)
	}

	if _, err := os.Stat(filepath.Join(downloadDir, "xklb-metadata.db")); err == nil {
		// The fake wrapper does not create the xklb database, but the
		// environment variable must point at the download directory.
		t.Log("xklb database was created by the fake wrapper")
	}
}

func TestResolve(t *testing.T) {
	downloadDir := t.TempDir()

	t.Run("auto prefers lb", func(t *testing.T) {
		dir := t.TempDir()
		writeFakeDownloader(t, dir, "lb")
		writeFakeDownloader(t, dir, "lb-wrapper")
		t.Setenv("PATH", dir)

		cfg, err := downloader.Resolve("auto", downloadDir)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Kind != downloader.KindLB || filepath.Base(cfg.Command) != "lb" {
			t.Errorf("config = %#v", cfg)
		}
	})

	t.Run("auto falls back to lb-wrapper", func(t *testing.T) {
		dir := t.TempDir()
		writeFakeDownloader(t, dir, "lb-wrapper")
		t.Setenv("PATH", dir)

		cfg, err := downloader.Resolve("auto", downloadDir)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Kind != downloader.KindWrapper {
			t.Errorf("kind = %q", cfg.Kind)
		}
	})

	t.Run("explicit command not found", func(t *testing.T) {
		if _, err := downloader.Resolve("does-not-exist", downloadDir); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestOptionsNormalize(t *testing.T) {
	tests := []struct {
		name    string
		options downloader.Options
		wantErr bool
	}{
		{name: "valid", options: downloader.Options{URL: "https://example.test/x", MediaType: "video"}},
		{name: "file scheme", options: downloader.Options{URL: "file:///etc/passwd"}, wantErr: true},
		{name: "missing scheme", options: downloader.Options{URL: "example.test/x"}, wantErr: true},
		{
			name:    "bad media type",
			options: downloader.Options{URL: "https://example.test/x", MediaType: "image"},
			wantErr: true,
		},
		{
			name:    "bad subtitles",
			options: downloader.Options{URL: "https://example.test/x", Subtitles: "maybe"},
			wantErr: true,
		},
		{
			name:    "bad audio format",
			options: downloader.Options{URL: "https://example.test/x", MediaType: "audio", AudioFormat: "wma"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.options.Normalize()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Normalize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManagerRejectsBusy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake downloader needs a POSIX shell")
	}
	dir := t.TempDir()
	slow := filepath.Join(dir, "lb")
	if err := os.WriteFile(slow, []byte("#!/bin/sh\nsleep 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	manager, err := downloader.NewManager(slow, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	if _, err := manager.Start(
		downloader.Options{URL: "https://example.test/x", Database: filepath.Join(dir, "x.db")},
	); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Start(
		downloader.Options{URL: "https://example.test/y", Database: filepath.Join(dir, "x.db")},
	); !errors.Is(err, downloader.ErrBusy) {
		t.Fatalf("second Start error = %v, want ErrBusy", err)
	}
	manager.Cancel()
}
