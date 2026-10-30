//go:build !linux && !windows

package sysaccent

// Read has no desktop accent to read here.
func Read() (string, bool) { return "", false }
