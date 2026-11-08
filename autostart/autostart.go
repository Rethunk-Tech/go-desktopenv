// Package autostart starts a program at sign-in, or stops doing so.
// Linux writes an XDG autostart desktop entry under the config home, or asks
// the Background portal inside a Flatpak. Windows sets a HKCU Run value.
// Elsewhere Enable, Disable and Enabled return ErrUnsupported.
package autostart

import "errors"

// ErrUnsupported is returned by Enable, Disable and Enabled on a platform that
// has no start-at-sign-in mechanism here.
var ErrUnsupported = errors.New("autostart: unsupported")
