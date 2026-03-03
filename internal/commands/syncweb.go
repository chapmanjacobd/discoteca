//go:build syncweb

package commands

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/chapmanjacobd/discotheque/internal/models"
	"github.com/chapmanjacobd/discotheque/internal/syncweb"
	"github.com/chapmanjacobd/discotheque/internal/utils"
	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/protocol"
)

type SyncwebCmd struct {
	models.CoreFlags
	models.SyncwebFlags

	Create    SyncwebCreateCmd    `cmd:"" help:"Create a syncweb folder" aliases:"init,in,share"`
	Join      SyncwebJoinCmd      `cmd:"" help:"Join syncweb folders/devices" aliases:"import,clone"`
	Accept    SyncwebAcceptCmd    `cmd:"" help:"Add a device to syncweb" aliases:"add"`
	Drop      SyncwebDropCmd      `cmd:"" help:"Remove a device from syncweb" aliases:"remove,reject"`
	Folders   SyncwebFoldersCmd   `cmd:"" help:"List Syncthing folders" aliases:"list-folders,lsf"`
	Devices   SyncwebDevicesCmd   `cmd:"" help:"List Syncthing devices" aliases:"list-devices,lsd"`
	Ls        SyncwebLsCmd        `cmd:"" help:"List files at the current directory level" aliases:"list"`
	Find      SyncwebFindCmd      `cmd:"" help:"Search for files by filename, size, and modified date" aliases:"fd,search"`
	Stat      SyncwebStatCmd      `cmd:"" help:"Display detailed file status information from Syncthing"`
	Sort      SyncwebSortCmd      `cmd:"" help:"Sort Syncthing files by multiple criteria"`
	Download  SyncwebDownloadCmd  `cmd:"" help:"Mark file paths for download/sync" aliases:"dl,upload,unignore,sync"`
	Automatic SyncwebAutomaticCmd `cmd:"" help:"Start syncweb-automatic daemon"`
	Start     SyncwebStartCmd     `cmd:"" help:"Start Syncweb" aliases:"restart"`
	Stop      SyncwebStopCmd      `cmd:"" help:"Shut down Syncweb" aliases:"shutdown,quit"`
	Version   SyncwebVersionCmd   `cmd:"" help:"Show Syncweb version"`
}

func (c *SyncwebCmd) AfterApply() error {
	if c.SyncwebHome == "" {
		c.SyncwebHome = filepath.Join(os.Getenv("HOME"), ".config", "syncweb")
	}
	return nil
}

func (c *SyncwebCmd) WithSyncweb(fn func(s *syncweb.Syncweb) error) error {
	s, err := syncweb.NewSyncweb(c.SyncwebHome, "disco-syncweb", "")
	if err != nil {
		return err
	}
	if err := s.Start(); err != nil {
		return err
	}
	defer s.Stop()
	return fn(s)
}

type SyncwebCreateCmd struct {
	Paths []string `arg:"" optional:"" default:"." help:"Path to folder"`
}

func (c *SyncwebCreateCmd) Run(g *SyncwebCmd) error {
	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		for _, p := range c.Paths {
			abs, _ := filepath.Abs(p)
			id := filepath.Base(abs) // Simplified folder ID generation
			err := s.AddFolder(id, id, abs, config.FolderTypeSendReceive)
			if err != nil {
				slog.Error("Failed to add folder", "path", abs, "error", err)
			} else {
				slog.Info("Added folder", "id", id, "path", abs)
			}
		}
		return nil
	})
}

type SyncwebJoinCmd struct {
	URLs   []string `arg:"" required:"" help:"Syncweb URLs (syncweb://folder-id#device-id)"`
	Prefix string   `help:"Path to parent folder" env:"SYNCWEB_HOME"`
}

func (c *SyncwebJoinCmd) Run(g *SyncwebCmd) error {
	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		for _, url := range c.URLs {
			// Basic parsing of syncweb://folder-id#device-id
			trimmed := strings.TrimPrefix(url, "syncweb://")
			parts := strings.SplitN(trimmed, "#", 2)
			if len(parts) != 2 {
				slog.Error("Invalid URL format", "url", url)
				continue
			}
			folderID := parts[0]
			deviceID := parts[1]

			if err := s.AddDevice(deviceID, deviceID, false); err != nil {
				slog.Error("Failed to add device", "id", deviceID, "error", err)
				continue
			}

			prefix := c.Prefix
			if prefix == "" {
				prefix = g.SyncwebHome
			}
			path := filepath.Join(prefix, folderID)
			if err := s.AddFolder(folderID, folderID, path, config.FolderTypeSendReceive); err != nil {
				slog.Error("Failed to add folder", "id", folderID, "error", err)
				continue
			}

			if err := s.AddFolderDevice(folderID, deviceID); err != nil {
				slog.Error("Failed to share folder with device", "folder", folderID, "device", deviceID, "error", err)
				continue
			}

			slog.Info("Joined syncweb", "folder", folderID, "device", deviceID)
		}
		return nil
	})
}

type SyncwebAcceptCmd struct {
	DeviceIDs  []string `arg:"" required:"" help:"Syncthing device IDs"`
	FolderIDs  []string `help:"Add devices to folders"`
	Introducer bool     `help:"Configure devices as introducers"`
}

func (c *SyncwebAcceptCmd) Run(g *SyncwebCmd) error {
	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		for _, devID := range c.DeviceIDs {
			if err := s.AddDevice(devID, devID, c.Introducer); err != nil {
				slog.Error("Failed to add device", "id", devID, "error", err)
				continue
			}
			for _, fldID := range c.FolderIDs {
				if err := s.AddFolderDevice(fldID, devID); err != nil {
					slog.Error("Failed to share folder with device", "folder", fldID, "device", devID, "error", err)
				}
			}
		}
		return nil
	})
}

type SyncwebDropCmd struct {
	DeviceIDs []string `arg:"" required:"" help:"Syncthing device IDs"`
	FolderIDs []string `help:"Remove devices from folders"`
}

func (c *SyncwebDropCmd) Run(g *SyncwebCmd) error {
	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		// Syncthing lib doesn't have a simple "DropDevice" in Cfg.Modify without more logic
		// For now, we'll just log that it's not fully implemented
		slog.Warn("Drop command not fully implemented in Go port yet")
		return nil
	})
}

type SyncwebFoldersCmd struct {
	Pending bool `help:"Show pending folders"`
	Join    bool `help:"Join pending folders"`
	Pause   bool `help:"Pause matching folders"`
	Resume  bool `help:"Resume matching folders"`
	Delete  bool `help:"Delete matching folders"`
}

func (c *SyncwebFoldersCmd) Run(g *SyncwebCmd) error {
	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		cfg := s.Node.Cfg.RawCopy()

		if c.Pending || c.Join {
			for _, dev := range cfg.Devices {
				pending, err := s.Node.App.Internals.PendingFolders(dev.DeviceID)
				if err != nil {
					continue
				}
				for folderID := range pending {
					fmt.Printf("Pending: %s from %s\n", folderID, dev.DeviceID)
					if c.Join {
						path := filepath.Join(g.SyncwebHome, folderID)
						if err := s.AddFolder(folderID, folderID, path, config.FolderTypeSendReceive); err != nil {
							slog.Error("Failed to join folder", "id", folderID, "error", err)
						} else {
							slog.Info("Joined folder", "id", folderID, "path", path)
							if err := s.AddFolderDevice(folderID, dev.DeviceID.String()); err != nil {
								slog.Error("Failed to share folder with source device", "folder", folderID, "device", dev.DeviceID, "error", err)
							}
						}
					}
				}
			}
			if !c.Join {
				if c.Pending {
					return nil
				}
			} else {
				// Refresh config after joining
				cfg = s.Node.Cfg.RawCopy()
			}
		}

		for _, f := range cfg.Folders {
			status := "OK"
			if f.Paused {
				status = "Paused"
			}
			fmt.Printf("%s: %s (%s) [%s]\n", f.ID, f.Label, f.Path, status)

			if c.Pause && !f.Paused {
				if err := s.PauseFolder(f.ID); err != nil {
					slog.Error("Failed to pause folder", "id", f.ID, "error", err)
				} else {
					slog.Info("Paused folder", "id", f.ID)
				}
			}
			if c.Resume && f.Paused {
				if err := s.ResumeFolder(f.ID); err != nil {
					slog.Error("Failed to resume folder", "id", f.ID, "error", err)
				} else {
					slog.Info("Resumed folder", "id", f.ID)
				}
			}
			if c.Delete {
				if err := s.DeleteFolder(f.ID); err != nil {
					slog.Error("Failed to delete folder", "id", f.ID, "error", err)
				} else {
					slog.Info("Deleted folder", "id", f.ID)
				}
			}
		}
		return nil
	})
}

type SyncwebDevicesCmd struct {
	Pending bool `help:"Show pending devices"`
	Accept  bool `help:"Accept pending devices"`
	Pause   bool `help:"Pause matching devices"`
	Resume  bool `help:"Resume matching devices"`
	Delete  bool `help:"Delete matching devices"`
}

func (c *SyncwebDevicesCmd) Run(g *SyncwebCmd) error {
	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		if c.Pending || c.Accept {
			pending := s.GetPendingDevices()
			for id, time := range pending {
				fmt.Printf("Pending: %s (since %v)\n", id, time)
				if c.Accept {
					if err := s.AddDevice(id, id, false); err != nil {
						slog.Error("Failed to auto-accept device", "id", id, "error", err)
					} else {
						slog.Info("Auto-accepted device", "id", id)
					}
				}
			}
		}

		cfg := s.Node.Cfg.RawCopy()
		for _, d := range cfg.Devices {
			status := "OK"
			if d.Paused {
				status = "Paused"
			}
			fmt.Printf("%s: %s [%s]\n", d.DeviceID, d.Name, status)

			if c.Pause && !d.Paused {
				if err := s.PauseDevice(d.DeviceID.String()); err != nil {
					slog.Error("Failed to pause device", "id", d.DeviceID, "error", err)
				} else {
					slog.Info("Paused device", "id", d.DeviceID)
				}
			}
			if c.Resume && d.Paused {
				if err := s.ResumeDevice(d.DeviceID.String()); err != nil {
					slog.Error("Failed to resume device", "id", d.DeviceID, "error", err)
				} else {
					slog.Info("Resumed device", "id", d.DeviceID)
				}
			}
			if c.Delete {
				if err := s.DeleteDevice(d.DeviceID.String()); err != nil {
					slog.Error("Failed to delete device", "id", d.DeviceID, "error", err)
				} else {
					slog.Info("Deleted device", "id", d.DeviceID)
				}
			}
		}
		return nil
	})
}

type SyncwebLsCmd struct {
	Paths         []string `arg:"" optional:"" default:"." help:"Path relative to the root"`
	Long          bool     `short:"l" help:"Use long listing format"`
	HumanReadable bool     `help:"Print sizes in human readable format" default:"true"`
}

func (c *SyncwebLsCmd) Run(g *SyncwebCmd) error {
	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		for _, p := range c.Paths {
			if p == "." || p == "" || p == "/" {
				// List all folders
				cfg := s.Node.Cfg.RawCopy()
				for _, f := range cfg.Folders {
					fmt.Printf("%s/ (%s)\n", f.ID, f.Path)
				}
				continue
			}

			var folderID string
			var prefix string

			if after, ok := strings.CutPrefix(p, "syncweb://"); ok {
				trimmed := after
				parts := strings.SplitN(trimmed, "/", 2)
				folderID = parts[0]
				if len(parts) > 1 {
					prefix = parts[1]
				}
			} else {
				// Try to find which folder this path belongs to
				abs, _ := filepath.Abs(p)
				cfg := s.Node.Cfg.RawCopy()
				for _, f := range cfg.Folders {
					if strings.HasPrefix(abs, f.Path) {
						folderID = f.ID
						prefix, _ = filepath.Rel(f.Path, abs)
						break
					}
				}
			}

			if folderID == "" {
				slog.Error("Path is not in a syncweb folder", "path", p)
				continue
			}

			if prefix != "" && !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}

			seq, cancel := s.Node.App.Internals.AllGlobalFiles(folderID)
			resultsMap := make(map[string]bool)

			if c.Long {
				fmt.Printf("%-4s %10s  %12s  %s\n", "Type", "Size", "Modified", "Name")
				fmt.Println(strings.Repeat("-", 40))
			}

			for meta := range seq {
				name := meta.Name
				if !strings.HasPrefix(name, prefix) || name == prefix {
					continue
				}

				rel := strings.TrimPrefix(name, prefix)
				parts := strings.Split(rel, "/")
				entryName := parts[0]
				isDir := len(parts) > 1

				if _, ok := resultsMap[entryName]; ok {
					continue
				}
				resultsMap[entryName] = true

				if isDir {
					if c.Long {
						fmt.Printf("d    %10s  %12s  %s/\n", "-", "", entryName)
					} else {
						fmt.Printf("%s/\n", entryName)
					}
				} else {
					if c.Long {
						sizeStr := fmt.Sprintf("%d", meta.Size)
						if c.HumanReadable {
							sizeStr = utils.FormatSize(meta.Size)
						}
						modTime := meta.ModTime().Format("02 Jan 15:04")
						fmt.Printf("-    %10s  %12s  %s\n", sizeStr, modTime, entryName)
					} else {
						fmt.Println(entryName)
					}
				}
			}
			cancel()
		}
		return nil
	})
}

type SyncwebFindCmd struct {
	Pattern  string   `arg:"" optional:"" default:".*" help:"Search patterns"`
	Type     string   `help:"Filter by type (f=file, d=directory)" short:"t"`
	FullPath bool     `help:"Search full path (default: filename only)" short:"p"`
	Paths    []string `arg:"" optional:"" help:"Root directories to search"`
}

func (c *SyncwebFindCmd) Run(g *SyncwebCmd) error {
	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		re, err := regexp.Compile("(?i)" + c.Pattern)
		if err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}

		cfg := s.Node.Cfg.RawCopy()
		for _, f := range cfg.Folders {
			seq, cancel := s.Node.App.Internals.AllGlobalFiles(f.ID)
			for meta := range seq {
				isDir := meta.Type == protocol.FileInfoTypeDirectory
				if c.Type == "f" && isDir {
					continue
				}
				if c.Type == "d" && !isDir {
					continue
				}

				searchTarget := meta.Name
				if !c.FullPath {
					searchTarget = filepath.Base(meta.Name)
				}

				if re.MatchString(searchTarget) {
					fmt.Printf("syncweb://%s/%s\n", f.ID, meta.Name)
				}
			}
			cancel()
		}
		return nil
	})
}

type SyncwebStatCmd struct {
	Paths []string `arg:"" required:"" help:"Files or directories to stat"`
}

func (c *SyncwebStatCmd) Run(g *SyncwebCmd) error {
	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		for _, p := range c.Paths {
			localPath, folderID, err := s.ResolveLocalPath(p)
			if err != nil {
				abs, _ := filepath.Abs(p)
				cfg := s.Node.Cfg.RawCopy()
				for _, f := range cfg.Folders {
					if strings.HasPrefix(abs, f.Path) {
						folderID = f.ID
						localPath = abs
						err = nil
						break
					}
				}
			}

			if err != nil || folderID == "" {
				slog.Error("Could not resolve path to a syncweb folder", "path", p)
				continue
			}

			rootPath, _ := s.GetFolderPath(folderID)
			relativePath, _ := filepath.Rel(rootPath, localPath)
			info, ok, err := s.GetGlobalFileInfo(folderID, relativePath)
			if err != nil {
				slog.Error("Failed to get file info", "path", p, "error", err)
				continue
			}
			if !ok {
				fmt.Printf("%s: Not found in cluster\n", p)
				continue
			}

			fmt.Printf("File: %s\n", info.Name)
			fmt.Printf("Size: %d bytes (%s)\n", info.Size, utils.FormatSize(info.Size))
			fmt.Printf("Modified: %v\n", info.ModTime())
			fmt.Printf("Type: %v\n", info.Type)
			fmt.Printf("Permissions: %o\n", info.Permissions)
			fmt.Printf("Blocks: %d\n", len(info.Blocks))
			fmt.Printf("Deleted: %v\n", info.Deleted)
			fmt.Printf("NoPermissions: %v\n", info.NoPermissions)
		}
		return nil
	})
}

type SyncwebSortCmd struct {
	Paths     []string `arg:"" optional:"" help:"File paths to sort"`
	Sort      []string `help:"Sort criteria (size, name)" default:"name"`
	LimitSize string   `help:"Stop after printing N bytes" short:"S"`
}

func (c *SyncwebSortCmd) Run(g *SyncwebCmd) error {
	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		limitBytes := int64(0)
		if c.LimitSize != "" {
			limitBytes, _ = utils.HumanToBytes(c.LimitSize)
		}

		type fileWithInfo struct {
			Path string
			Info protocol.FileInfo
		}
		var files []fileWithInfo

		for _, p := range c.Paths {
			localPath, folderID, err := s.ResolveLocalPath(p)
			if err != nil {
				abs, _ := filepath.Abs(p)
				cfg := s.Node.Cfg.RawCopy()
				for _, f := range cfg.Folders {
					if strings.HasPrefix(abs, f.Path) {
						folderID = f.ID
						localPath = abs
						err = nil
						break
					}
				}
			}

			if err != nil || folderID == "" {
				continue
			}

			rootPath, _ := s.GetFolderPath(folderID)
			relativePath, _ := filepath.Rel(rootPath, localPath)
			info, ok, err := s.GetGlobalFileInfo(folderID, relativePath)
			if err == nil && ok {
				files = append(files, fileWithInfo{Path: p, Info: info})
			}
		}

		sort.Slice(files, func(i, j int) bool {
			for _, criterion := range c.Sort {
				reverse := strings.HasPrefix(criterion, "-")
				if reverse {
					criterion = criterion[1:]
				}

				var less bool
				switch criterion {
				case "size":
					less = files[i].Info.Size < files[j].Info.Size
				case "name":
					less = files[i].Info.Name < files[j].Info.Name
				default:
					continue
				}

				if files[i].Info.Size == files[j].Info.Size && criterion == "size" {
					continue
				}
				if files[i].Info.Name == files[j].Info.Name && criterion == "name" {
					continue
				}

				if reverse {
					return !less
				}
				return less
			}
			return false
		})

		currentSize := int64(0)
		for _, f := range files {
			if limitBytes > 0 && currentSize+f.Info.Size > limitBytes {
				break
			}
			fmt.Println(f.Path)
			currentSize += f.Info.Size
		}
		return nil
	})
}

type SyncwebDownloadCmd struct {
	Paths []string `arg:"" optional:"" help:"File or directory paths to download"`
}

func (c *SyncwebDownloadCmd) Run(g *SyncwebCmd) error {
	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		type downloadItem struct {
			folderID string
			relPath  string
			size     int64
		}
		var items []downloadItem
		var totalSize int64

		for _, p := range c.Paths {
			localPath, folderID, err := s.ResolveLocalPath(p)
			if err != nil {
				// Try to resolve as local path if it doesn't have syncweb:// prefix
				abs, _ := filepath.Abs(p)
				cfg := s.Node.Cfg.RawCopy()
				for _, f := range cfg.Folders {
					if strings.HasPrefix(abs, f.Path) {
						folderID = f.ID
						localPath = abs
						err = nil
						break
					}
				}
			}

			if err != nil || folderID == "" {
				slog.Error("Could not resolve path to a syncweb folder", "path", p)
				continue
			}

			rootPath, _ := s.GetFolderPath(folderID)
			relativePath, _ := filepath.Rel(rootPath, localPath)
			info, ok, err := s.GetGlobalFileInfo(folderID, relativePath)
			if err != nil || !ok {
				slog.Error("Failed to get file info for download", "path", p)
				continue
			}

			items = append(items, downloadItem{folderID, relativePath, info.Size})
			totalSize += info.Size
		}

		if len(items) == 0 {
			fmt.Println("No files found to download")
			return nil
		}

		fmt.Printf("\nDownload Summary:\n")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("%-20s %-30s %10s\n", "Folder ID", "Path", "Size")
		fmt.Println(strings.Repeat("-", 60))
		for _, item := range items {
			fmt.Printf("%-20s %-30s %10s\n", item.folderID, item.relPath, utils.FormatSize(item.size))
		}
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("TOTAL: %d files (%s)\n", len(items), utils.FormatSize(totalSize))

		if !g.Yes {
			if !utils.Confirm(fmt.Sprintf("Mark %d files for download?", len(items))) {
				fmt.Println("Download cancelled")
				return nil
			}
		}

		for _, item := range items {
			if err := s.Unignore(item.folderID, item.relPath); err != nil {
				slog.Error("Failed to trigger download", "folder", item.folderID, "path", item.relPath, "error", err)
			} else {
				slog.Info("Download triggered", "folder", item.folderID, "path", item.relPath)
			}
		}
		return nil
	})
}

type SyncwebAutomaticCmd struct {
	Devices bool `help:"Auto-accept devices"`
	Folders bool `help:"Auto-join folders"`
	Local   bool `default:"true" help:"Only auto-accept local devices"`
}

func (c *SyncwebAutomaticCmd) Run(g *SyncwebCmd) error {
	slog.Info("Starting syncweb-automatic", "devices", c.Devices, "folders", c.Folders, "localOnly", c.Local)

	return g.WithSyncweb(func(s *syncweb.Syncweb) error {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			if c.Devices {
				pending := s.GetPendingDevices()
				for id := range pending {
					slog.Info("Auto-accepting device", "id", id)
					if err := s.AddDevice(id, id, false); err != nil {
						slog.Error("Failed to auto-accept device", "id", id, "error", err)
					}
				}
			}

			if c.Folders {
				cfg := s.Node.Cfg.RawCopy()
				for _, dev := range cfg.Devices {
					pending, _ := s.Node.App.Internals.PendingFolders(dev.DeviceID)
					for folderID := range pending {
						slog.Info("Auto-joining folder", "id", folderID, "from", dev.DeviceID)
						path := filepath.Join(g.SyncwebHome, folderID)
						if err := s.AddFolder(folderID, folderID, path, config.FolderTypeSendReceive); err == nil {
							s.AddFolderDevice(folderID, dev.DeviceID.String())
						}
					}
				}
			}

			<-ticker.C
		}
	})
}

type SyncwebStartCmd struct{}

func (c *SyncwebStartCmd) Run(g *SyncwebCmd) error {
	slog.Info("Syncweb starts automatically when used via CLI or Serve")
	return nil
}

type SyncwebStopCmd struct{}

func (c *SyncwebStopCmd) Run(g *SyncwebCmd) error {
	slog.Info("Syncweb stops automatically when CLI or Serve exits")
	return nil
}

type SyncwebVersionCmd struct{}

func (c *SyncwebVersionCmd) Run(g *SyncwebCmd) error {
	fmt.Println("Syncweb (Go port) v0.0.1")
	return nil
}
