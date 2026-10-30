package utils

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"slices"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/models"
)

// RandomString returns a random hexadecimal string of the given length
func RandomString(n int) string {
	b := make([]byte, n/2+1)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)[:n]
}

// FilterPath checks if a path matches PathFilterFlags
func FilterPath(path string, flags models.PathFilterFlags) bool {
	if len(flags.Paths) > 0 {
		matched := slices.Contains(flags.Paths, path)
		if !matched {
			return false
		}
	}

	if len(flags.Include) > 0 {
		matched := false
		for _, pattern := range flags.Include {
			if strings.Contains(path, pattern) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(flags.Exclude) > 0 {
		for _, pattern := range flags.Exclude {
			if strings.Contains(path, pattern) {
				return false
			}
		}
	}

	if flags.Regex != "" {
		if matched, _ := regexp.MatchString(flags.Regex, path); !matched {
			return false
		}
	}

	if len(flags.PathContains) > 0 {
		for _, s := range flags.PathContains {
			if !strings.Contains(path, s) {
				return false
			}
		}
	}

	return true
}
