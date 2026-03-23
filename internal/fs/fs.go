package fs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func FindMedia(root string, filter map[string]bool) (map[string]os.FileInfo, error) {
	files := make(map[string]os.FileInfo)
	ch := make(chan struct {
		Path string
		Info os.FileInfo
	})

	var walkErr error
	go func() {
		defer close(ch)
		walkErr = FindMediaChan(root, filter, ch)
	}()

	for res := range ch {
		files[res.Path] = res.Info
	}
	return files, walkErr
}

func FindMediaChan(root string, filter map[string]bool, ch chan<- struct {
	Path string
	Info os.FileInfo
}) error {
	allowed := filter
	if allowed == nil {
		allowed = utils.MediaExtensionMap
	}

	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if allowed[ext] {
			info, err := d.Info()
			if err != nil {
				return nil // Skip files we can't access
			}
			ch <- struct {
				Path string
				Info os.FileInfo
			}{Path: path, Info: info}
		}
		return nil
	})
}

