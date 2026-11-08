//go:build unix

package autostart

import (
	"context"
	"errors"
	"os"

	"github.com/Rethunk-Tech/go-desktopenv/desktopentry"
	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-desktopenv/portal"
	"github.com/Rethunk-Tech/go-desktopenv/userdirs"
)

// Enable arranges for argv to run at sign-in under the name app. Inside a
// Flatpak it asks the Background portal; otherwise it writes app.desktop in
// the XDG autostart folder. argv is the command the session runs, or the
// command the portal starts inside the sandbox. ctx bounds the portal's consent dialog.
func Enable(ctx context.Context, app string, argv []string) error {
	return apply(ctx, app, argv, true)
}

// Disable removes the start-at-sign-in entry named app.
func Disable(ctx context.Context, app string) error { return apply(ctx, app, nil, false) }

// Enabled reports whether a start-at-sign-in entry named app exists in the
// XDG autostart folder. Inside a Flatpak the portal's entry is outside the
// sandbox's view, so Enabled reports false there; keep the choice yourself.
func Enabled(app string) (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	root, err := os.OpenRoot(userdirs.ConfigHome(home))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer func() { _ = root.Close() }()
	sub, err := root.OpenRoot("autostart")
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer func() { _ = sub.Close() }()
	_, err = sub.Stat(app + ".desktop")
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func apply(ctx context.Context, app string, argv []string, enable bool) error {
	if enable && len(argv) == 0 {
		return errors.New("autostart: empty command")
	}
	if flatpak.Inside() {
		// The sandbox cannot write the host's autostart folder; the Background portal writes the entry for it.
		return portal.SetAutostart(ctx, portal.Autostart{Enable: enable, Command: argv})
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfg := userdirs.ConfigHome(home)
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		return err
	}
	root, err := os.OpenRoot(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	if enable {
		if err := root.Mkdir("autostart", 0o700); err != nil && !os.IsExist(err) {
			return err
		}
		sub, err := root.OpenRoot("autostart")
		if err != nil {
			return err
		}
		defer func() { _ = sub.Close() }()
		return sub.WriteFile(app+".desktop", []byte(entry(app, argv)), 0o600)
	}
	sub, err := root.OpenRoot("autostart")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = sub.Close() }()
	err = sub.Remove(app + ".desktop")
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func entry(app string, argv []string) string {
	return "[Desktop Entry]\nType=Application\nName=" + desktopentry.Value(app) + "\nExec=" + desktopentry.Exec(argv...) + "\nX-GNOME-Autostart-enabled=true\n"
}
