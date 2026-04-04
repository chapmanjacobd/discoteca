package utils

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Logger is a struct-based wrapper around slog.Logger
// It provides a cleaner API and makes dependency injection easier
type Logger struct {
	log *slog.Logger
}

// NewLogger creates a new Logger instance with the given handler
func NewLogger(handler slog.Handler) *Logger {
	return &Logger{
		log: slog.New(handler),
	}
}

// NewDefaultLogger creates a Logger with PlainHandler using the specified level
func NewDefaultLogger(level slog.Leveler, out io.Writer) *Logger {
	return NewLogger(&PlainHandler{
		Level: level,
		Out:   out,
	})
}

// Info logs an informational message with key-value pairs
func (l *Logger) Info(msg string, args ...any) {
	l.log.Info(msg, args...)
}

// Debug logs a debug message with key-value pairs
func (l *Logger) Debug(msg string, args ...any) {
	l.log.Debug(msg, args...)
}

// Warn logs a warning message with key-value pairs
func (l *Logger) Warn(msg string, args ...any) {
	l.log.Warn(msg, args...)
}

// Error logs an error message with key-value pairs
func (l *Logger) Error(msg string, args ...any) {
	l.log.Error(msg, args...)
}

// With returns a new Logger that includes the given key-value pairs in all log messages
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		log: l.log.With(args...),
	}
}

// Enabled checks if the given level is enabled
func (l *Logger) Enabled(level slog.Level) bool {
	return l.log.Enabled(context.Background(), level)
}

// PlainHandler is a custom slog.Handler that outputs plain key=value format
type PlainHandler struct {
	Level slog.Leveler
	Out   io.Writer
	Attrs []slog.Attr
}

func (h *PlainHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.Level.Level()
}

func (h *PlainHandler) Handle(_ context.Context, r slog.Record) error {
	var msg strings.Builder
	msg.WriteString(r.Message)
	for _, a := range h.Attrs {
		fmt.Fprintf(&msg, "\n    %s=%v", a.Key, a.Value.Any())
	}
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(&msg, "\n    %s=%v", a.Key, a.Value.Any())
		return true
	})
	_, err := fmt.Fprintln(h.Out, msg.String())
	return err
}

func (h *PlainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PlainHandler{
		Level: h.Level,
		Out:   h.Out,
		Attrs: append(h.Attrs, attrs...),
	}
}

func (h *PlainHandler) WithGroup(_ string) slog.Handler {
	// Not implementing groups for now
	return h
}
