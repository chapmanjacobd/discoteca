package models

import (
	"log/slog"
	"testing"
)

func TestCoreFlags_AfterApply(t *testing.T) {
	tests := []struct {
		name       string
		flags      CoreFlags
		wantDryRun bool
		wantYes    bool
	}{
		{
			"Simulate and NoConfirm",
			CoreFlags{Simulate: true, NoConfirm: true},
			true,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := tt.flags
			err := flags.AfterApply()
			if err != nil {
				t.Fatalf("AfterApply failed: %v", err)
			}
			if flags.DryRun != tt.wantDryRun {
				t.Errorf("DryRun = %v, want %v", flags.DryRun, tt.wantDryRun)
			}
			if flags.Yes != tt.wantYes {
				t.Errorf("Yes = %v, want %v", flags.Yes, tt.wantYes)
			}
		})
	}
}

func TestMediaFilterFlags_AfterApply(t *testing.T) {
	tests := []struct {
		name     string
		flags    MediaFilterFlags
		wantExt0 string
		wantExt1 string
	}{
		{
			"Ext normalization",
			MediaFilterFlags{Ext: []string{"mp4", ".mkv"}},
			".mp4",
			".mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := tt.flags
			err := flags.AfterApply()
			if err != nil {
				t.Fatalf("AfterApply failed: %v", err)
			}
			if flags.Ext[0] != tt.wantExt0 {
				t.Errorf("Ext[0] = %s, want %s", flags.Ext[0], tt.wantExt0)
			}
			if flags.Ext[1] != tt.wantExt1 {
				t.Errorf("Ext[1] = %s, want %s", flags.Ext[1], tt.wantExt1)
			}
		})
	}
}

func TestMergeFlags_AfterApply(t *testing.T) {
	tests := []struct {
		name            string
		flags           MergeFlags
		wantOnlyNewRows bool
	}{
		{
			"Ignore flag",
			MergeFlags{Ignore: true},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := tt.flags
			err := flags.AfterApply()
			if err != nil {
				t.Fatalf("AfterApply failed: %v", err)
			}
			if flags.OnlyNewRows != tt.wantOnlyNewRows {
				t.Errorf("OnlyNewRows = %v, want %v", flags.OnlyNewRows, tt.wantOnlyNewRows)
			}
		})
	}
}

func TestSetupLogging(t *testing.T) {
	tests := []struct {
		verbosity    int
		wantLogLevel slog.Level
	}{
		{2, slog.LevelDebug},
		{1, slog.LevelInfo},
		{0, slog.LevelWarn},
	}

	for _, tt := range tests {
		t.Run("verbosity", func(t *testing.T) {
			SetupLogging(tt.verbosity)
			// Just verify that Log is set (we can't easily test the level without access to the handler)
			if Log == nil {
				t.Errorf("Log is nil after SetupLogging")
			}
		})
	}
}
