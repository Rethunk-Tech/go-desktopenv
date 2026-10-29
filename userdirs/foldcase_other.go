//go:build !windows

package userdirs

func foldCase(p string) string { return p }
