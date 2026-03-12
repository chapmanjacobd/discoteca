//go:build !pyroscope

package main

// setupProfiling is a no-op when not built with pyroscope tag
func setupProfiling() func() {
	return func() {}
}
