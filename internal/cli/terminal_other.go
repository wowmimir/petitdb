//go:build !windows && !linux && !darwin

package cli

import "fmt"

// Stub for unsupported Unix platforms – raw mode not available,
// but CLI will still work in line‑buffered mode.
func enableRawMode(fd int) (interface{}, error) {
	return nil, fmt.Errorf("raw terminal not supported on this platform")
}

func disableRawMode(fd int, state interface{}) {
	// nothing
}