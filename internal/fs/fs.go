package fs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func FindMedia(root string, filter map[string]bool) (map[string]os.FileInfo, error) {
	files := make(map[string]os.FileInfo)
	ch := make(chan FindMediaResult)

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

type FindMediaResult struct {
	Path       string
	Info       os.FileInfo
	FilesCount int
	DirsCount  int
}

func FindMediaChan(root string, filter map[string]bool, ch chan<- FindMediaResult) error {
	allowed := filter
	if allowed == nil {
		allowed = utils.MediaExtensionMap
	}

	var filesCount, dirsCount int

	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirsCount++
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if allowed[ext] {
			info, err := d.Info()
			if err != nil {
				return nil // Skip files we can't access
			}
			filesCount++
			ch <- FindMediaResult{Path: path, Info: info, FilesCount: filesCount, DirsCount: dirsCount}
		}
		return nil
	})
}
