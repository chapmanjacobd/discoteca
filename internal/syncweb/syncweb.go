package syncweb

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	stmodel "github.com/syncthing/syncthing/lib/model"
	"github.com/syncthing/syncthing/lib/protocol"
)

type Measurement struct {
	TotalTime time.Duration
	Count     int64
	Errors    int64
}

type DeviceInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Addresses  []string `json:"addresses"`
	Introducer bool     `json:"introducer"`
	Paused     bool     `json:"paused"`
}

type FolderInfo struct {
	ID      string   `json:"id"`
	Label   string   `json:"label"`
	Path    string   `json:"path"`
	Type    string   `json:"type"`
	Paused  bool     `json:"paused"`
	Devices []string `json:"devices"`
}

type Measurements struct {
	mutex sync.RWMutex
	data  map[protocol.DeviceID]*Measurement
}

func NewMeasurements() *Measurements {
	return &Measurements{
		data: make(map[protocol.DeviceID]*Measurement),
	}
}

func (m *Measurements) Record(id protocol.DeviceID, duration time.Duration, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.data[id]; !ok {
		m.data[id] = &Measurement{}
	}
	meas := m.data[id]
	if err != nil {
		meas.Errors++
	} else {
		meas.TotalTime += duration
		meas.Count++
	}
}

func (m *Measurements) Score(id protocol.DeviceID) float64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	meas, ok := m.data[id]
	if !ok || meas.Count == 0 {
		return 0 // Neutral score for new peers
	}

	avgTime := float64(meas.TotalTime) / float64(meas.Count)
	errorRate := float64(meas.Errors) / float64(meas.Count+meas.Errors)

	// Lower is better. Penalty for errors.
	return avgTime * (1.0 + errorRate*10.0)
}

type Syncweb struct {
	Node         *Node
	Measurements *Measurements
}

func NewSyncweb(homeDir string, name string, listenAddr string) (*Syncweb, error) {
	node, err := NewNode(homeDir, name, listenAddr)
	if err != nil {
		return nil, err
	}

	return &Syncweb{
		Node:         node,
		Measurements: NewMeasurements(),
	}, nil
}

func (s *Syncweb) Start() error {
	return s.Node.Start()
}

func (s *Syncweb) Stop() {
	s.Node.Stop()
}

func (s *Syncweb) IsRunning() bool {
	return s.Node.IsRunning()
}

// AddDevice adds a device to the Syncthing configuration
func (s *Syncweb) AddDevice(deviceID string, name string, introducer bool) error {
	id, err := protocol.DeviceIDFromString(deviceID)
	if err != nil {
		return err
	}

	_, err = s.Node.Cfg.Modify(func(cfg *config.Configuration) {
		for i, dev := range cfg.Devices {
			if dev.DeviceID == id {
				cfg.Devices[i].Name = name
				cfg.Devices[i].Introducer = introducer
				return
			}
		}
		device := cfg.Defaults.Device.Copy()
		device.DeviceID = id
		device.Name = name
		device.Introducer = introducer
		device.Addresses = []string{"dynamic"}
		cfg.SetDevice(device)
	})
	return err
}

// SetDeviceAddresses sets explicit addresses for a device
func (s *Syncweb) SetDeviceAddresses(deviceID string, addresses []string) error {
	id, err := protocol.DeviceIDFromString(deviceID)
	if err != nil {
		return err
	}

	_, err = s.Node.Cfg.Modify(func(cfg *config.Configuration) {
		for i, dev := range cfg.Devices {
			if dev.DeviceID == id {
				cfg.Devices[i].Addresses = addresses
				return
			}
		}
	})
	return err
}

// AddFolder adds a folder to the Syncthing configuration
func (s *Syncweb) AddFolder(id string, label string, path string, folderType config.FolderType) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(absPath, 0o700); err != nil {
		return err
	}

	_, err = s.Node.Cfg.Modify(func(cfg *config.Configuration) {
		if _, _, ok := cfg.Folder(id); ok {
			return // Already exists
		}
		fld := cfg.Defaults.Folder.Copy()
		fld.ID = id
		fld.Label = label
		fld.Path = absPath
		fld.Type = folderType
		cfg.SetFolder(fld)
	})
	return err
}

// AddFolderDevice shares a folder with a device
func (s *Syncweb) AddFolderDevice(folderID string, deviceID string) error {
	devID, err := protocol.DeviceIDFromString(deviceID)
	if err != nil {
		return err
	}

	_, err = s.Node.Cfg.Modify(func(cfg *config.Configuration) {
		for i, fld := range cfg.Folders {
			if fld.ID == folderID {
				for _, dev := range fld.Devices {
					if dev.DeviceID == devID {
						return // Already shared
					}
				}
				cfg.Folders[i].Devices = append(cfg.Folders[i].Devices, config.FolderDeviceConfiguration{
					DeviceID: devID,
				})
				return
			}
		}
	})
	return err
}

// AddFolderDevices shares a folder with multiple devices
func (s *Syncweb) AddFolderDevices(folderID string, deviceIDs []string) error {
	ids := make([]protocol.DeviceID, 0, len(deviceIDs))
	for _, did := range deviceIDs {
		id, err := protocol.DeviceIDFromString(did)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	_, err := s.Node.Cfg.Modify(func(cfg *config.Configuration) {
		for i, fld := range cfg.Folders {
			if fld.ID == folderID {
				existing := make(map[protocol.DeviceID]bool)
				for _, dev := range fld.Devices {
					existing[dev.DeviceID] = true
				}

				for _, id := range ids {
					if !existing[id] {
						cfg.Folders[i].Devices = append(cfg.Folders[i].Devices, config.FolderDeviceConfiguration{
							DeviceID: id,
						})
					}
				}
				return
			}
		}
	})
	return err
}

func (s *Syncweb) PauseFolder(id string) error {
	_, err := s.Node.Cfg.Modify(func(cfg *config.Configuration) {
		for i, f := range cfg.Folders {
			if f.ID == id {
				cfg.Folders[i].Paused = true
				return
			}
		}
	})
	return err
}

func (s *Syncweb) ResumeFolder(id string) error {
	_, err := s.Node.Cfg.Modify(func(cfg *config.Configuration) {
		for i, f := range cfg.Folders {
			if f.ID == id {
				cfg.Folders[i].Paused = false
				return
			}
		}
	})
	return err
}

func (s *Syncweb) DeleteFolder(id string) error {
	_, err := s.Node.Cfg.Modify(func(cfg *config.Configuration) {
		for i, f := range cfg.Folders {
			if f.ID == id {
				cfg.Folders = append(cfg.Folders[:i], cfg.Folders[i+1:]...)
				return
			}
		}
	})
	return err
}

func (s *Syncweb) PauseDevice(id string) error {
	devID, err := protocol.DeviceIDFromString(id)
	if err != nil {
		return err
	}
	_, err = s.Node.Cfg.Modify(func(cfg *config.Configuration) {
		for i, d := range cfg.Devices {
			if d.DeviceID == devID {
				cfg.Devices[i].Paused = true
				return
			}
		}
	})
	return err
}

func (s *Syncweb) ResumeDevice(id string) error {
	devID, err := protocol.DeviceIDFromString(id)
	if err != nil {
		return err
	}
	_, err = s.Node.Cfg.Modify(func(cfg *config.Configuration) {
		for i, d := range cfg.Devices {
			if d.DeviceID == devID {
				cfg.Devices[i].Paused = false
				return
			}
		}
	})
	return err
}

func (s *Syncweb) DeleteDevice(id string) error {
	devID, err := protocol.DeviceIDFromString(id)
	if err != nil {
		return err
	}
	_, err = s.Node.Cfg.Modify(func(cfg *config.Configuration) {
		for i, d := range cfg.Devices {
			if d.DeviceID == devID {
				cfg.Devices = append(cfg.Devices[:i], cfg.Devices[i+1:]...)
				return
			}
		}
	})
	return err
}

// GetFolders returns a list of folder information
func (s *Syncweb) GetFolders() []FolderInfo {
	cfg := s.Node.Cfg.RawCopy()
	folders := make([]FolderInfo, 0, len(cfg.Folders))
	for _, f := range cfg.Folders {
		devices := make([]string, 0, len(f.Devices))
		for _, d := range f.Devices {
			devices = append(devices, d.DeviceID.String())
		}
		folders = append(folders, FolderInfo{
			ID:      f.ID,
			Label:   f.Label,
			Path:    f.Path,
			Type:    f.Type.String(),
			Paused:  f.Paused,
			Devices: devices,
		})
	}
	return folders
}

// GetDevices returns a list of device information
func (s *Syncweb) GetDevices() []DeviceInfo {
	cfg := s.Node.Cfg.RawCopy()
	devices := make([]DeviceInfo, 0, len(cfg.Devices))
	for _, d := range cfg.Devices {
		devices = append(devices, DeviceInfo{
			ID:         d.DeviceID.String(),
			Name:       d.Name,
			Addresses:  d.Addresses,
			Introducer: d.Introducer,
			Paused:     d.Paused,
		})
	}
	return devices
}

// ResolveLocalPath resolves a syncweb:// URL to a local filesystem path,
// ensuring the path is within the folder's root directory.
func (s *Syncweb) ResolveLocalPath(syncwebPath string) (string, string, error) {
	if !strings.HasPrefix(syncwebPath, "syncweb://") {
		return "", "", fmt.Errorf("invalid syncweb path: %s", syncwebPath)
	}

	trimmed := strings.TrimPrefix(syncwebPath, "syncweb://")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid syncweb path: %s", syncwebPath)
	}

	folderID := parts[0]
	relativePath := filepath.Clean(parts[1])

	if strings.HasPrefix(relativePath, "..") || filepath.IsAbs(relativePath) {
		return "", "", fmt.Errorf("invalid relative path: %s", relativePath)
	}

	cfg := s.Node.Cfg.RawCopy()
	for _, f := range cfg.Folders {
		if f.ID == folderID {
			fullPath := filepath.Join(f.Path, relativePath)
			// Final safety check: ensure the joined path is still within f.Path
			rel, err := filepath.Rel(f.Path, fullPath)
			if err != nil || strings.HasPrefix(rel, "..") {
				return "", "", fmt.Errorf("traversal detected: %s", relativePath)
			}
			return fullPath, folderID, nil
		}
	}

	return "", "", fmt.Errorf("folder not found: %s", folderID)
}

// GetFolderPath returns the local path for a folder ID (internal use only)
func (s *Syncweb) GetFolderPath(folderID string) (string, bool) {
	cfg := s.Node.Cfg.RawCopy()
	for _, f := range cfg.Folders {
		if f.ID == folderID {
			return f.Path, true
		}
	}
	return "", false
}

// Unignore removes a file from the ignore list by adding an unignore (!) pattern
func (s *Syncweb) Unignore(folderID, relativePath string) error {
	lines, _, err := s.Node.App.Internals.Ignores(folderID)
	if err != nil {
		return err
	}

	pattern := "!" + relativePath
	if slices.Contains(lines, pattern) {
		return nil // Already unignored
	}

	lines = append(lines, pattern)
	return s.Node.App.Internals.SetIgnores(folderID, lines)
}

// GetGlobalFileInfo returns information about a file across the cluster
func (s *Syncweb) GetGlobalFileInfo(folderID, path string) (protocol.FileInfo, bool, error) {
	return s.Node.App.Internals.GlobalFileInfo(folderID, path)
}

// SyncwebReadSeeker implements io.ReadSeeker by fetching blocks from Syncthing peers
type SyncwebReadSeeker struct {
	s        *Syncweb
	folderID string
	info     protocol.FileInfo
	offset   int64
	ctx      context.Context
}

func (s *Syncweb) NewReadSeeker(ctx context.Context, folderID, path string) (*SyncwebReadSeeker, error) {
	info, ok, err := s.GetGlobalFileInfo(folderID, path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("file not found in cluster: %s", path)
	}

	return &SyncwebReadSeeker{
		s:        s,
		folderID: folderID,
		info:     info,
		offset:   0,
		ctx:      ctx,
	}, nil
}

func (r *SyncwebReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = r.offset + offset
	case io.SeekEnd:
		newOffset = r.info.Size + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newOffset < 0 {
		return 0, fmt.Errorf("negative offset: %d", newOffset)
	}
	r.offset = newOffset
	return r.offset, nil
}

func (r *SyncwebReadSeeker) Read(p []byte) (n int, err error) {
	if r.offset >= r.info.Size {
		return 0, io.EOF
	}

	wantedSize := int64(len(p))
	if r.offset+wantedSize > r.info.Size {
		wantedSize = r.info.Size - r.offset
	}

	if wantedSize <= 0 {
		return 0, io.EOF
	}

	// Calculate which blocks we need
	blockSize := int64(r.info.BlockSize())
	startBlock := r.offset / blockSize
	endBlock := (r.offset + wantedSize - 1) / blockSize

	var totalRead int64
	for i := startBlock; i <= endBlock; i++ {
		block := r.info.Blocks[i]

		// Determine which peers have this block
		availables, err := r.s.Node.App.Internals.BlockAvailability(r.folderID, r.info, block)
		if err != nil {
			return int(totalRead), err
		}
		if len(availables) == 0 {
			return int(totalRead), fmt.Errorf("no peers available for block %d", i)
		}

		// Sort available peers by their performance score (lower is better)
		slices.SortFunc(availables, func(a, b stmodel.Availability) int {
			scoreA := r.s.Measurements.Score(a.ID)
			scoreB := r.s.Measurements.Score(b.ID)
			if scoreA < scoreB {
				return -1
			}
			if scoreA > scoreB {
				return 1
			}
			return 0
		})

		var data []byte
		var downloadErr error
		for _, peer := range availables {
			startTime := time.Now()
			data, downloadErr = r.s.Node.App.Internals.DownloadBlock(r.ctx, peer.ID, r.folderID, r.info.Name, int(i), block, peer.FromTemporary)
			r.s.Measurements.Record(peer.ID, time.Since(startTime), downloadErr)
			if downloadErr == nil {
				break
			}
			slog.Warn("Failed to download block from peer, trying next", "peer", peer.ID, "error", downloadErr)
		}

		if downloadErr != nil {
			return int(totalRead), fmt.Errorf("all peers failed to provide block %d: %w", i, downloadErr)
		}

		// Calculate how much of this block we actually need
		blockOffset := r.offset + totalRead - block.Offset
		dataStart := max(blockOffset, 0)

		dataEnd := int64(len(data))
		remainingNeeded := wantedSize - totalRead
		if dataEnd-dataStart > remainingNeeded {
			dataEnd = dataStart + remainingNeeded
		}

		copied := copy(p[totalRead:], data[dataStart:dataEnd])
		totalRead += int64(copied)
	}

	r.offset += totalRead
	return int(totalRead), nil
}
