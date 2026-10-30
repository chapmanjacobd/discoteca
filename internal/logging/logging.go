// Package logging provides the process-wide structured logger used by all
// discoteca packages.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/chapmanjacobd/discoteca/internal/db"
)

// Logger is the structured logging interface used throughout the application.
type Logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// Log is the process-wide logger. It is replaced by [SetupLogging].
var Log Logger = newLogger(slog.LevelWarn)

// SetupLogging configures the process-wide logger for the given verbosity.
func SetupLogging(verbosity int) {
	// Create handler with appropriate level
	var level slog.Level
	if verbosity >= 2 {
		level = slog.LevelDebug
		db.SetDebugMode(true)
	} else if verbosity == 1 {
		level = slog.LevelInfo
	} else {
		// Default to Warn (hides Info and Debug, shows Warn and Error)
		level = slog.LevelWarn
	}

	Log = newLogger(level)
}

func newLogger(level slog.Level) Logger {
	handler := &plainHandler{
		level: level,
		out:   os.Stderr,
	}
	return &slogLogger{slog.New(handler)}
}

// plainHandler implements [slog.Handler] with plain output.
type plainHandler struct {
	level slog.Level
	out   io.Writer
	attrs []slog.Attr
}

func (h *plainHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *plainHandler) Handle(_ context.Context, r slog.Record) error {
	var msg strings.Builder
	msg.WriteString(r.Message)
	for _, a := range h.attrs {
		fmt.Fprintf(&msg, "\n    %s=%v", a.Key, a.Value.Any())
	}
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(&msg, "\n    %s=%v", a.Key, a.Value.Any())
		return true
	})
	_, err := fmt.Fprintln(h.out, msg.String())
	return err
}

func (h *plainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &plainHandler{
		level: h.level,
		out:   h.out,
		attrs: append(h.attrs, attrs...),
	}
}

func (h *plainHandler) WithGroup(_ string) slog.Handler {
	return h
}

// slogLogger wraps [slog.Logger] and implements Logger interface.
type slogLogger struct {
	*slog.Logger
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{l.Logger.With(args...)}
}
