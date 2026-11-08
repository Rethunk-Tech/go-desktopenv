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

// Enable arranges for e.Argv to run at sign-in. Inside a Flatpak it asks the
// Background portal; otherwise it writes e.ID.desktop in the XDG autostart
// folder. ctx bounds the portal's consent dialog.
func Enable(ctx context.Context, e Entry) error {
	return apply(ctx, e, true)
}

// Disable removes the start-at-sign-in entry id.
func Disable(ctx context.Context, id string) error { return apply(ctx, Entry{ID: id}, false) }

// Enabled reports whether the start-at-sign-in entry id exists in the
// XDG autostart folder. Inside a Flatpak the portal's entry is outside the
// sandbox's view, so Enabled reports false there; keep the choice yourself.
func Enabled(id string) (bool, error) {
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
	_, err = sub.Stat(id + ".desktop")
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func apply(ctx context.Context, e Entry, enable bool) error {
	if enable && len(e.Argv) == 0 {
		return errors.New("autostart: empty command")
	}
	if flatpak.Inside() {
		// The sandbox cannot write the host's autostart folder; the Background portal writes the entry for it.
		return portal.SetAutostart(ctx, portal.Autostart{Enable: enable, Command: e.Argv, Reason: e.Reason})
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
		return sub.WriteFile(e.ID+".desktop", []byte(entry(e)), 0o600)
	}
	sub, err := root.OpenRoot("autostart")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer func() { _ = sub.Close() }()
	err = sub.Remove(e.ID + ".desktop")
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// name is what the desktop entry shows: Name, else the ID.
func (e Entry) name() string {
	if e.Name != "" {
		return e.Name
	}
	return e.ID
}

func entry(e Entry) string {
	return "[Desktop Entry]\nType=Application\nName=" + desktopentry.Value(e.name()) + "\nExec=" + desktopentry.Exec(e.Argv...) + "\nX-GNOME-Autostart-enabled=true\n"
}
