package utils

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/chapmanjacobd/discotheque/internal/models"
)

// ShellQuote returns a shell-escaped version of the string
func ShellQuote(s string) string {
	if s == "" {
		return "''"
	}
	// If it doesn't contain any special characters, return it as is
	safeChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_./"
	allSafe := true
	for _, r := range s {
		if !strings.ContainsRune(safeChars, r) {
			allSafe = false
			break
		}
	}
	if allSafe {
		return s
	}

	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// RenameMoveFile moves a file from src to dst, creating parent directories if needed
func RenameMoveFile(flags models.GlobalFlags, src, dst string) error {
	if flags.Simulate {
		fmt.Printf("mv %s %s\n", src, dst)
		return nil
	}

	err := os.Rename(src, dst)
	if err != nil {
		// If dst parent doesn't exist
		if os.IsNotExist(err) {
			parent := filepath.Dir(dst)
			if err := os.MkdirAll(parent, 0o755); err != nil {
				return err
			}
			return os.Rename(src, dst)
		}

		// Cross-device move fallback
		if strings.Contains(err.Error(), "invalid cross-device link") {
			// Basic copy and delete
			input, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			err = os.WriteFile(dst, input, 0o644)
			if err != nil {
				return err
			}
			return os.Remove(src)
		}
	}
	return err
}

// RenameNoReplace renames a file only if the destination does not exist
func RenameNoReplace(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("the destination file %s already exists", dst)
	}
	return os.Rename(src, dst)
}

// Trash moves a file to the system trash if available, otherwise deletes it
func Trash(flags models.GlobalFlags, path string) error {
	if !FileExists(path) {
		return nil
	}

	if flags.Simulate {
		fmt.Printf("trash %s\n", path)
		return nil
	}

	trashCmd := "trash-put" // Prefer trash-put from trash-cli
	err := CmdDetach(trashCmd, path)
	if err != nil {
		// Try 'trash' if 'trash-put' fails
		err = CmdDetach("trash", path)
	}

	if err != nil {
		slog.Debug("trash command failed, unlinking instead", "path", path, "error", err)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove file %s: %w", path, err)
		}
	}
	return nil
}

// FlattenWrapperFolder removes a single subfolder if it's the only entry in outputDir
func FlattenWrapperFolder(outputDir string) error {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return err
	}

	// Filter out hidden files
	var visibleEntries []os.DirEntry
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			visibleEntries = append(visibleEntries, e)
		}
	}

	if len(visibleEntries) == 1 && visibleEntries[0].IsDir() {
		wrapperName := visibleEntries[0].Name()
		wrapperPath := filepath.Join(outputDir, wrapperName)
		slog.Info("Flattening wrapper folder", "folder", wrapperName)

		subEntries, err := os.ReadDir(wrapperPath)
		if err != nil {
			return err
		}

		var conflictItem string
		for _, se := range subEntries {
			name := se.Name()
			if name == wrapperName {
				conflictItem = name
				continue
			}

			src := filepath.Join(wrapperPath, name)
			dst := filepath.Join(outputDir, name)
			if err := os.Rename(src, dst); err != nil {
				// Try fallback
				if err := RenameMoveFile(models.GlobalFlags{}, src, dst); err != nil {
					return err
				}
			}
		}

		if conflictItem != "" {
			src := filepath.Join(wrapperPath, conflictItem)
			tempDst := filepath.Join(outputDir, conflictItem+".tmp")
			if err := os.Rename(src, tempDst); err != nil {
				if err := RenameMoveFile(models.GlobalFlags{}, src, tempDst); err != nil {
					return err
				}
			}
			if err := os.Remove(wrapperPath); err != nil {
				return err
			}
			return os.Rename(tempDst, filepath.Join(outputDir, conflictItem))
		}

		return os.Remove(wrapperPath)
	}
	return nil
}

// ResolveAbsolutePath expands user home and returns absolute path if it exists
func ResolveAbsolutePath(s string) string {
	if strings.HasPrefix(s, "~") {
		home, _ := os.UserHomeDir()
		s = filepath.Join(home, s[1:])
	}

	abs, err := filepath.Abs(s)
	if err == nil {
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}
	return s
}
