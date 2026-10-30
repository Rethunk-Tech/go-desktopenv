//go:build !linux && !windows

package fonts

// List has no font source here.
func List() ([]string, error) { return nil, nil }
