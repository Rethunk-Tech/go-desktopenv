// Package schemehandler registers a program as the user's handler of URL schemes, hands each scheme back to the
// handler it replaced, and forwards a URL to that handler. On Linux it writes a user desktop entry and sets the
// x-scheme-handler defaults through xdg-mime, on the host from inside a Flatpak; on Windows it writes the scheme keys
// under HKCU\Software\Classes. Remembering each scheme's previous Owner.ID between runs is the caller's job.
package schemehandler

import "errors"

// ErrUnsupported is returned by every call on a system that is neither Linux nor Windows.
var ErrUnsupported = errors.New("schemehandler: URL scheme registration is not supported on this system")

// Config is what to register. Exe and Schemes apply everywhere; the rest is Linux only.
type Config struct {
	// Exe is the program a URL is handed to, as its first argument.
	Exe string
	// Schemes are every scheme the program may claim, without "://". The desktop entry lists them all while any is
	// claimed, and Restore hands them all back.
	Schemes []string
	// DesktopID is the entry's file name, "org.example.App.desktop". An entry whose name starts with the id without
	// its extension is the app's own (a profile shortcut, say), never an AppImage integrator's.
	DesktopID string
	// Entry is the rest of the desktop entry.
	Entry Entry
	// Packaged means a package installed DesktopID system-wide: no user entry is written or looked for, and that
	// entry is the handler. Inside a Flatpak it is always true.
	Packaged bool
}

// Entry is the desktop entry's keys besides Exec and MimeType. An empty Comment, Icon, StartupWMClass or list is
// left out.
type Entry struct {
	Name, Comment, Icon  string
	Categories, Keywords []string
	StartupWMClass       string
	// MimeTypes are listed after the schemes whether or not any scheme is claimed, such as the app's own file type.
	MimeTypes []string
}

// Owner is the program the system opens a scheme's URLs with. ID is what Restore and Release take back: a desktop
// entry id on Linux, the saved open command on Windows. Name is the program's display name or path. Mine is whether
// it is this Config's program. An empty ID means no program owns the scheme.
type Owner struct {
	ID   string
	Name string
	Mine bool
}

var errNoPrevious = errors.New("schemehandler: no previous handler")
