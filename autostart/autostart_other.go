//go:build !unix && !windows

package autostart

import "context"

// Enable returns ErrUnsupported.
func Enable(context.Context, Entry) error { return ErrUnsupported }

// Disable returns ErrUnsupported.
func Disable(context.Context, string) error { return ErrUnsupported }

// Enabled returns ErrUnsupported.
func Enabled(string) (bool, error) { return false, ErrUnsupported }
