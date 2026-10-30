// Package hostfile reaches the user's desktop configuration (mimeapps.list, browser manifests) where it lives: the
// plain file call outside a Flatpak, a host sh script through flatpak-spawn inside one, which is not granted those
// files.
package hostfile

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-desktopenv/hostenv"
	"github.com/Rethunk-Tech/go-desktopenv/userdirs"
	"github.com/Rethunk-Tech/go-safefs"
)

// Runner runs a command and returns its standard output.
type Runner func(name string, args ...string) (string, error)

// Exec is the Runner that starts a real process, with the AppImage's variables removed from its environment.
func Exec(name string, args ...string) (string, error) {
	cmd := exec.CommandContext(context.Background(), name, args...) //nolint:gosec // G204: callers run fixed desktop tools or a handler the user registered
	cmd.Env = hostenv.Environ()
	out, err := cmd.Output()
	return string(out), err
}

// Host runs the file steps through Run.
type Host struct{ Run Runner }

// Command runs name where the user's desktop lives: directly, or from inside a Flatpak through flatpak-spawn, where
// name is looked up on the host's PATH.
func (h Host) Command(name string, args ...string) (string, error) {
	if flatpak.Inside() {
		hostName, hostArgs := flatpak.HostArgv("", nil, filepath.Base(name), args...)
		return h.Run(hostName, hostArgs...)
	}
	return h.Run(name, args...)
}

// missing is the exit status Read's script gives a missing file.
const missing = 3

// ConfigHome is the host's XDG config folder; a Flatpak's own XDG_CONFIG_HOME points into its sandbox.
func ConfigHome(home string) string {
	if flatpak.Inside() {
		return userdirs.HostConfigHome(home)
	}
	return userdirs.ConfigHome(home)
}

// Resolve follows a symlink (a dotfile manager's), so a rewrite lands in its target and the link stays.
func (h Host) Resolve(path string) string {
	if flatpak.Inside() {
		if out, err := h.Command("sh", "-c", `readlink -f "$1"`, "sh", path); err == nil && out != "" {
			return strings.TrimSuffix(out, "\n")
		}
		return path
	}
	if target, err := filepath.EvalSymlinks(path); err == nil {
		return target
	}
	return path
}

// Read is the file's bytes, or fs.ErrNotExist.
func (h Host) Read(path string) ([]byte, error) {
	if !flatpak.Inside() {
		return safefs.ReadFile(path)
	}
	out, err := h.Command("sh", "-c", `[ -e "$1" ] || exit 3; cat "$1"`, "sh", path)
	if exit := (*exec.ExitError)(nil); errors.As(err, &exit) && exit.ExitCode() == missing {
		return nil, fs.ErrNotExist
	}
	return []byte(out), err
}

// Write replaces the file whole, creating its folder.
func (h Host) Write(path string, b []byte, perm fs.FileMode) error {
	if !flatpak.Inside() {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		return safefs.WriteFileAtomic(path, b, perm)
	}
	_, err := h.Command("sh", "-c", `mkdir -p "$(dirname "$1")" && printf %s "$2" >"$1.new" && mv "$1.new" "$1"`,
		"sh", path, string(b))
	return err
}

// Remove deletes the file; a missing one is no error.
func (h Host) Remove(path string) error {
	if !flatpak.Inside() {
		if err := safefs.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	_, err := h.Command("sh", "-c", `rm -f "$1"`, "sh", path)
	return err
}

// IsDir reports whether path is a folder.
func (h Host) IsDir(path string) bool {
	if !flatpak.Inside() {
		return safefs.IsDir(path)
	}
	_, err := h.Command("sh", "-c", `[ -d "$1" ]`, "sh", path)
	return err == nil
}
