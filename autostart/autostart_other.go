//go:build !unix && !windows

package autostart

// Enable returns ErrUnsupported.
func Enable(string, []string) error { return ErrUnsupported }

// Disable returns ErrUnsupported.
func Disable(string) error { return ErrUnsupported }

// Enabled returns ErrUnsupported.
func Enabled(string) (bool, error) { return false, ErrUnsupported }
