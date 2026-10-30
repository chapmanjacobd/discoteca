package db

// InitFtsConfig is retained as a startup hook for callers. FTS support is
// determined at build time by the fts5 build tag; there is no runtime toggle.
func InitFtsConfig() {}

// IsFtsEnabled reports whether this binary was built with FTS5 support.
func IsFtsEnabled() bool {
	return FtsEnabled
}
