package fs

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/charlievieth/fastwalk"
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
	info, err := os.Stat(root)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		if filter != nil {
			ext := strings.ToLower(filepath.Ext(root))
			if !filter[ext] {
				return nil
			}
		}
		ch <- FindMediaResult{
			Path:       root,
			Info:       info,
			FilesCount: 1,
			DirsCount:  0,
		}
		return nil
	}

	var filesCount int64
	var dirsCount int64

	conf := fastwalk.Config{
		Follow: false,
	}

	return fastwalk.Walk(&conf, root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			atomic.AddInt64(&dirsCount, 1)
			return nil
		}

		if filter != nil {
			ext := strings.ToLower(filepath.Ext(path))
			if !filter[ext] {
				return nil
			}
		}

		i, err := d.Info()
		if err != nil {
			return nil // Skip files we can't access
		}

		fc := atomic.AddInt64(&filesCount, 1)
		dc := atomic.LoadInt64(&dirsCount)
		ch <- FindMediaResult{
			Path:       path,
			Info:       i,
			FilesCount: int(fc),
			DirsCount:  int(dc),
		}
		return nil
	})
}
