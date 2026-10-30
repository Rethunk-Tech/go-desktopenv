//go:build !windows

package lnk

import "errors"

// ErrUnsupported is returned by Write, Arguments and Target off Windows.
var ErrUnsupported = errors.New("lnk: shortcut files are Windows only")

// Write is Windows only.
func Write(string, Shortcut) error { return ErrUnsupported }

// Arguments is Windows only.
func Arguments(string) (string, error) { return "", ErrUnsupported }

// Target is Windows only.
func Target(string) (string, error) { return "", ErrUnsupported }
