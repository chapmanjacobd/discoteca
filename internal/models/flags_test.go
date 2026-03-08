package models

import (
	"log/slog"
	"testing"
)

func TestCoreFlags_AfterApply(t *testing.T) {
	flags := CoreFlags{
		Simulate:  true,
		NoConfirm: true,
	}
	err := flags.AfterApply()
	if err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}
	if !flags.DryRun {
		t.Error("Expected DryRun to be true when Simulate is true")
	}
	if !flags.Yes {
		t.Error("Expected Yes to be true when NoConfirm is true")
	}
}

func TestMediaFilterFlags_AfterApply(t *testing.T) {
	flags := MediaFilterFlags{
		Ext: []string{"mp4", ".mkv"},
	}
	err := flags.AfterApply()
	if err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}
	if flags.Ext[0] != ".mp4" {
		t.Errorf("Expected .mp4, got %s", flags.Ext[0])
	}
	if flags.Ext[1] != ".mkv" {
		t.Errorf("Expected .mkv, got %s", flags.Ext[1])
	}
}

func TestMergeFlags_AfterApply(t *testing.T) {
	flags := MergeFlags{
		Ignore: true,
	}
	err := flags.AfterApply()
	if err != nil {
		t.Fatalf("AfterApply failed: %v", err)
	}
	if !flags.OnlyNewRows {
		t.Error("Expected OnlyNewRows to be true when Ignore is true")
	}
}

func TestSetupLogging(t *testing.T) {
	SetupLogging(true)
	if LogLevel.Level() != slog.LevelDebug {
		t.Errorf("Expected debug level, got %v", LogLevel.Level())
	}

	SetupLogging(false)
	if LogLevel.Level() != slog.LevelInfo {
		t.Errorf("Expected info level, got %v", LogLevel.Level())
	}
}
