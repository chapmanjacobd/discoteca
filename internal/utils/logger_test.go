package utils_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/chapmanjacobd/discoteca/internal/utils"
)

func TestPlainHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := &utils.PlainHandler{
		Level: slog.LevelInfo,
		Out:   &buf,
	}

	logger := slog.New(handler)

	// Test Enabled
	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Info level should be enabled")
	}
	if handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Debug level should not be enabled")
	}

	// Test Handle
	logger.Info("test message", "key", "value")
	got := buf.String()
	if !strings.Contains(got, "test message") {
		t.Errorf("Expected message not found: %q", got)
	}
	if !strings.Contains(got, "key=value") {
		t.Errorf("Expected attribute not found: %q", got)
	}

	// Test WithAttrs
	buf.Reset()
	loggerWithAttrs := logger.With("fixed", "attr")
	loggerWithAttrs.Info("msg")
	got = buf.String()
	if !strings.Contains(got, "fixed=attr") {
		t.Errorf("Expected fixed attribute not found: %q", got)
	}

	// Test WithGroup (it just returns the same handler)
	loggerWithGroup := logger.WithGroup("group")
	buf.Reset()
	loggerWithGroup.Info("msg")
	if !strings.Contains(buf.String(), "msg") {
		t.Error("WithGroup should still log messages")
	}
}
