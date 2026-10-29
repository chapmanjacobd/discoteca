package db

import "sync/atomic"

// defaultCacheSizeMB is the historical SQLite page cache size. It is generous
// for a desktop but large for a single-board computer, so the serve command
// exposes it as a flag.
const defaultCacheSizeMB = 256

var cacheSizeMB atomic.Int64

func init() {
	cacheSizeMB.Store(defaultCacheSizeMB)
}

// SetCacheSizeMB sets the SQLite page cache size in MiB for connections opened
// after this call. Values below 1 restore the default.
func SetCacheSizeMB(mb int) {
	if mb < 1 {
		mb = defaultCacheSizeMB
	}
	cacheSizeMB.Store(int64(mb))
}

// CacheSizeMB returns the configured SQLite page cache size in MiB.
func CacheSizeMB() int {
	return int(cacheSizeMB.Load())
}
