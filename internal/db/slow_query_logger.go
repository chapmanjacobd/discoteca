package db

import (
	"sync/atomic"
	"time"
)

const slowQueryThreshold = 50 * time.Millisecond

// debugModeEnabled is an atomic flag to control slow query logging
var debugModeEnabled atomic.Bool

// SetDebugMode enables or disables debug mode for slow query logging
func SetDebugMode(enabled bool) {
	debugModeEnabled.Store(enabled)
}

// IsDebugMode returns true if debug mode is enabled
func IsDebugMode() bool {
	return debugModeEnabled.Load()
}
