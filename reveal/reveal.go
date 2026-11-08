// Package reveal shows a path in the system file manager.
package reveal

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/hostenv"
	"github.com/Rethunk-Tech/go-safefs"
)

// checkOpenable refuses a path the system handler should not be given: one that is not absolute (a leading dash would
// read as an option, and a relative name depends on the working folder), a network share path, or a symbolic link to
// a file, which would open whatever it points at. A link to a folder is how a data folder moved elsewhere looks, so it
// opens.
func checkOpenable(path string) error {
	if !filepath.IsAbs(path) || isUNC(path) {
		return errors.New("not a local absolute path")
	}
	link, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if link.Mode()&os.ModeSymlink != 0 {
		if !safefs.IsDir(path) {
			return errors.New("a link to a file is not opened")
		}
	}
	return nil
}

// Open shows path, a folder or a file, in the system file manager without waiting for it.
func Open(path string) error {
	if err := checkOpenable(path); err != nil {
		return err
	}
	name := "xdg-open"
	switch runtime.GOOS {
	case "windows":
		name = "explorer"
	case "darwin":
		name = "open"
	}
	cmd := exec.CommandContext(context.Background(), name, path) //nolint:gosec // G204: name is the OS file manager; path is validated
	cmd.Env = hostenv.Environ()
	setArgs(cmd, path)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// isUNC reports a network share path, in either the plain or the \\?\UNC\ spelling.
func isUNC(path string) bool {
	if strings.HasPrefix(strings.ToUpper(path), `\\?\UNC\`) {
		return true
	}
	return strings.HasPrefix(path, `\\`) && !strings.HasPrefix(path, `\\?\`)
}
