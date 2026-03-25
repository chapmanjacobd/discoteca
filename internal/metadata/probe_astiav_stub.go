//go:build !astiav

package metadata

import "fmt"

// newAstiavBackend is a stub that returns an error when astiav is not available
func newAstiavBackend() (ProbeBackend, error) {
	return nil, fmt.Errorf("astiav backend not available: build with -tags astiav and FFmpeg 8.0 dev libraries")
}
