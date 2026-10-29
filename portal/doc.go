// Package portal calls the xdg-desktop-portal interfaces a Flatpak app uses instead of writing host folders: the
// Background portal for starting at sign-in, DynamicLauncher for desktop launchers, Secret for a per-app master
// secret, and Settings for desktop appearance values. Every call opens the session bus, makes one request and closes
// it. Portal dialogs wait on the user, so every call takes a context: set a deadline, since none is imposed here.
// Off Linux every call returns ErrUnsupported.
package portal
