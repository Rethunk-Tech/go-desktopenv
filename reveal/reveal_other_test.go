//go:build !windows

package reveal

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenStartsTheFileManager(t *testing.T) {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	// A script, not a symlink to true: a coreutils multicall binary (Ubuntu 26.04's) refuses to
	// run under another name.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil { //nolint:gosec // G306: the fake must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if err := Open(t.TempDir()); err != nil {
		t.Fatal(err)
	}
}
