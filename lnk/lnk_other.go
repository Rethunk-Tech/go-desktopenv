//go:build !windows

package lnk

import "errors"

// ErrUnsupported is returned by Write and Arguments off Windows.
var ErrUnsupported = errors.New("lnk: shortcut files are Windows only")

// Write is Windows only.
func Write(string, Shortcut) error { return ErrUnsupported }

// Arguments is Windows only.
func Arguments(string) (string, error) { return "", ErrUnsupported }
