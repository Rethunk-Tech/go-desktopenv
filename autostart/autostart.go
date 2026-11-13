// Package autostart starts a program at sign-in, or stops doing so.
// Linux writes an XDG autostart desktop entry under the config home, or asks
// the Background portal inside a Flatpak. Windows sets a HKCU Run value.
// Elsewhere Enable, Disable and Enabled return ErrUnsupported.
package autostart

import (
	"errors"
	"strings"
)

// ErrUnsupported is returned by Enable, Disable and Enabled on a platform that
// has no start-at-sign-in mechanism here.
var ErrUnsupported = errors.New("autostart: unsupported")

func checkID(id string) error {
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, "/\\\x00") {
		return errors.New("autostart: ID is not a single path element")
	}
	return nil
}

// quoteProgram is the program of a Windows HKCU Run command line: always quoted, with quotes and trailing
// backslashes escaped so they cannot close the string.
func quoteProgram(p string) string {
	var b strings.Builder
	b.WriteByte('"')
	bs := 0
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case '\\':
			bs++
		case '"':
			b.WriteString(strings.Repeat(`\`, 2*bs+1))
			b.WriteByte('"')
			bs = 0
		default:
			b.WriteString(strings.Repeat(`\`, bs))
			b.WriteByte(p[i])
			bs = 0
		}
	}
	b.WriteString(strings.Repeat(`\`, 2*bs))
	b.WriteByte('"')
	return b.String()
}

// Entry is one program started at sign-in.
type Entry struct {
	// ID names the entry: the desktop file's stem on Linux, the Run value's name on Windows. Disable and Enabled take it.
	ID string
	// Name is what the session's list of startup programs shows; empty shows ID.
	Name string
	// Reason is shown in the Flatpak Background portal's permission dialog; empty sends none.
	Reason string
	// Argv is the command the session runs at sign-in, or inside a Flatpak the command the portal starts in the sandbox.
	Argv []string
}
