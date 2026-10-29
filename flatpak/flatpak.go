// Package flatpak detects a Flatpak sandbox and runs commands on the host from inside one. The sandbox cannot see
// the host's programs, processes or autostart folder, so a command reaches them through flatpak-spawn, which needs
// the org.freedesktop.Flatpak bus name (--talk-name=org.freedesktop.Flatpak) in the app's finish-args.
package flatpak

import (
	"context"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"
)

// Inside reports whether the process runs in a Flatpak sandbox (FLATPAK_ID is set).
func Inside() bool { return os.Getenv("FLATPAK_ID") != "" }

// HostArgv returns the flatpak-spawn command line that runs name on the host, in dir when non-empty, with env
// added. Pure: callers that start the process themselves (with their own pipes, process group, context) use it.
func HostArgv(dir string, env []string, name string, args ...string) (string, []string) {
	out := []string{"--host"}
	if dir != "" {
		out = append(out, "--directory="+dir)
	}
	for _, e := range env {
		out = append(out, "--env="+e)
	}
	out = append(out, name)
	return "flatpak-spawn", append(out, args...)
}

// cancelGrace is how long a cancelled flatpak-spawn gets to forward SIGTERM and exit before Go gives up on it.
const cancelGrace = 5 * time.Second

// CommandContext is exec.CommandContext, except that for name "flatpak-spawn" a cancel sends SIGTERM and WaitDelay
// is 5 s. flatpak-spawn forwards SIGTERM to the host child but cannot forward the SIGKILL Go sends by default, which
// would leave the child running.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // G204: callers pass a resolved binary or flatpak-spawn
	if name == "flatpak-spawn" {
		cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
		cmd.WaitDelay = cancelGrace
	}
	return cmd
}

// Command returns the Cmd that runs name on the host: flatpak-spawn --host inside a Flatpak, name itself outside.
// Inside, the Cmd is flatpak-spawn's, so Dir and Env set on it do not reach the host program; use HostArgv for
// those. It is built with CommandContext.
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	if Inside() {
		name, args = HostArgv("", nil, name, args...)
	}
	return CommandContext(ctx, name, args...)
}

// HostLookPath finds name on the host's PATH, or returns exec.ErrNotFound. It asks the host's sh rather than which,
// which minimal hosts do not install. It always spawns flatpak-spawn, so call it only inside a Flatpak.
func HostLookPath(ctx context.Context, name string) (string, error) {
	prog, args := HostArgv("", nil, "sh", "-c", `command -v -- "$1"`, "sh", name)
	out, err := CommandContext(ctx, prog, args...).Output()
	if err != nil {
		return "", exec.ErrNotFound
	}
	// command -v prints a builtin or alias by its bare name, which is not a path. The host is Linux, so its paths
	// are slash paths whatever OS this process runs on.
	p := strings.TrimSpace(string(out))
	if !path.IsAbs(p) {
		return "", exec.ErrNotFound
	}
	return p, nil
}
