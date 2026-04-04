package db

import (
	"log/slog"
	"sync/atomic"
	"time"
)

// SlowQueryThreshold is the minimum duration for a query to be considered slow
const SlowQueryThreshold = 50 * time.Millisecond

// debugModeEnabled is an atomic flag to control slow query logging
var debugModeEnabled atomic.Bool

// Log is the logger instance used by the db package. Set via SetLogger.
// Defaults to the global slog logger if not set.
var Log Logger = (*defaultLogger)(nil)

// Logger is a minimal logging interface to avoid direct slog dependencies
type Logger interface {
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// SetLogger sets the logger instance for the db package
func SetLogger(logger Logger) {
	Log = logger
}

// defaultLogger wraps the global slog.Logger as a Logger implementation
type defaultLogger struct{}

func (d *defaultLogger) Info(msg string, args ...any)  { slog.Info(msg, args...) }
func (d *defaultLogger) Debug(msg string, args ...any) { slog.Debug(msg, args...) }
func (d *defaultLogger) Warn(msg string, args ...any)  { slog.Warn(msg, args...) }
func (d *defaultLogger) Error(msg string, args ...any) { slog.Error(msg, args...) }

// SetDebugMode enables or disables debug mode for slow query logging
func SetDebugMode(enabled bool) {
	debugModeEnabled.Store(enabled)
}

// IsDebugMode returns true if debug mode is enabled
func IsDebugMode() bool {
	return debugModeEnabled.Load()
}
