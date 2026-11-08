//go:build windows

package autostart

import (
	"errors"
	"strings"

	winreg "golang.org/x/sys/windows/registry"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// Enable writes argv as a HKCU Run value named app, so the session starts it
// at sign-in.
func Enable(app string, argv []string) error {
	return apply(app, argv, true)
}

// Disable removes the HKCU Run value named app.
func Disable(app string) error { return apply(app, nil, false) }

// Enabled reports whether a HKCU Run value named app exists.
func Enabled(app string) (bool, error) {
	key, err := winreg.OpenKey(winreg.CURRENT_USER, runKey, winreg.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, winreg.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer func() { _ = key.Close() }()
	_, _, err = key.GetStringValue(app)
	if err != nil {
		if errors.Is(err, winreg.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func apply(app string, argv []string, enable bool) error {
	if enable && len(argv) == 0 {
		return errors.New("autostart: empty command")
	}
	key, _, err := winreg.CreateKey(winreg.CURRENT_USER, runKey, winreg.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = key.Close() }()
	if !enable {
		err := key.DeleteValue(app)
		if err != nil && !errors.Is(err, winreg.ErrNotExist) {
			return err
		}
		return nil
	}
	return key.SetStringValue(app, command(argv))
}

func command(argv []string) string {
	s := `"` + argv[0] + `"`
	if len(argv) == 1 {
		return s
	}
	return s + " " + strings.Join(argv[1:], " ")
}
