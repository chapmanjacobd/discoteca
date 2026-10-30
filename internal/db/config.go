package db

import (
	"sync"
	"sync/atomic"
)

// ftsEnabledConfig is a runtime configuration to enable/disable FTS.
// FTS is enabled by default; opt-out is reserved for non-FTS builds and tests.
var (
	ftsEnabledConfig  = func() *atomic.Bool { var b atomic.Bool; b.Store(true); return &b }()
	ftsConfigInitOnce sync.Once
)

// InitFtsConfig initializes the FTS configuration.
// This should be called early in the application lifecycle. FTS is enabled by
// default, so the call is idempotent and only guarantees a deterministic state.
func InitFtsConfig() {
	ftsConfigInitOnce.Do(func() {
		ftsEnabledConfig.Store(true)
	})
}

// IsFtsEnabled returns true if FTS is enabled both at compile time AND runtime
func IsFtsEnabled() bool {
	return FtsEnabled && ftsEnabledConfig.Load()
}
