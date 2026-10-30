package utils_test

import (
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestGetMpvWatchSocket(t *testing.T) {
	got := utils.GetMpvWatchSocket()
	if got == "" {
		t.Errorf("GetMpvWatchSocket returned empty string")
	}
}

func TestGetMpvWatchLaterDir(t *testing.T) {
	got := utils.GetMpvWatchLaterDir()
	if got == "" {
		t.Errorf("GetMpvWatchLaterDir returned empty string")
	}
}

func TestGetDirs(t *testing.T) {
	tests := []struct {
		name string
		fn   func() string
	}{
		{"GetCattNowPlayingFile", utils.GetCattNowPlayingFile},
		{"GetConfigDir", utils.GetConfigDir},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(); got == "" {
				t.Errorf("%s returned empty string", tt.name)
			}
		})
	}
}
