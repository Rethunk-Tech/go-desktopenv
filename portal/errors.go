package portal

import "errors"

// Declared without a build tag so every platform's build has the same errors.
var (
	// ErrCancelled is returned when the user dismisses the portal's dialog.
	ErrCancelled = errors.New("portal: cancelled by the user")
	// ErrNotAllowed is returned when autostart was asked for and the portal did not grant it.
	ErrNotAllowed = errors.New("portal: request was not allowed")
	// ErrNotFound is returned by ReadSetting when the portal has no such setting.
	ErrNotFound = errors.New("portal: setting not found")
	// ErrUnsupported is returned by every call on a platform without desktop portals.
	ErrUnsupported = errors.New("portal: desktop portals exist only on Linux")
)

// Autostart is a request to start Command at sign-in, or to stop doing so.
type Autostart struct {
	Enable  bool
	Command []string // run inside the sandbox at sign-in
	Reason  string   // shown in the permission dialog; empty sends none
}

// Launcher is a desktop launcher for the DynamicLauncher portal.
type Launcher struct {
	ID         string   // must start with the app id and end in .desktop (portal rule)
	Name       string   // proposed; the user may rename it in the dialog
	IconPNG    []byte   // the launcher's icon
	Exec       []string // argv, quoted and escaped by desktopentry.Exec
	Comment    string   // optional
	Categories []string // optional
}
