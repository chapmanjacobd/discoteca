package fs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func FindMedia(root string, filter map[string]bool) (map[string]os.FileInfo, error) {
	files := make(map[string]os.FileInfo)

	allowed := filter
	if allowed == nil {
		allowed = utils.MediaExtensionMap
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
			files[path] = info
		}
		return nil
	})
	return files, err
}
