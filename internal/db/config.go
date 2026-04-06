package db

import (
	"sync"
	"sync/atomic"
)

// ftsEnabledConfig is a runtime configuration to enable/disable FTS
// It defaults to true, but can be disabled via SetFtsEnabled(false)
// This is separate from the compile-time FtsEnabled constant
var (
	ftsEnabledConfig  atomic.Bool
	ftsConfigInitOnce sync.Once
)

// InitFtsConfig initializes the FTS configuration.
// This should be called early in the application lifecycle.
func InitFtsConfig() {
	ftsConfigInitOnce.Do(func() {
		ftsEnabledConfig.Store(false)
	})
}

// SetFtsEnabled sets the runtime FTS enabled state
func SetFtsEnabled(enabled bool) {
	ftsEnabledConfig.Store(enabled)
}

// IsFtsEnabled returns true if FTS is enabled both at compile time AND runtime
func IsFtsEnabled() bool {
	return FtsEnabled && ftsEnabledConfig.Load()
}
