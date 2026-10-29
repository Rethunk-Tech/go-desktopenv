// Package userdirs answers where a user's config and data folders are for a given home. Each function takes the
// home folder the caller resolves for (a test's temporary home, another account's profile).
//
// An environment value is used only when home is this process's own home (os.UserHomeDir, compared after
// filepath.Clean and, on Windows, case-folded) and the value is an absolute path; the XDG Base Directory spec says to
// ignore a relative one. Otherwise the default under home is returned. home == "" returns "" for every function,
// since the environment describes only the real home: the caller turns "" into an error. No function reads
// runtime.GOOS: the caller picks the layout it is resolving.
package userdirs

import (
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
)

// ConfigHome is $XDG_CONFIG_HOME, else home/.config.
func ConfigHome(home string) string { return pick("XDG_CONFIG_HOME", home, ".config") }

// DataHome is $XDG_DATA_HOME, else home/.local/share.
func DataHome(home string) string { return pick("XDG_DATA_HOME", home, ".local", "share") }

// AppData is %APPDATA%, else home/AppData/Roaming.
func AppData(home string) string { return pick("APPDATA", home, "AppData", "Roaming") }

// LocalAppData is %LOCALAPPDATA%, else home/AppData/Local.
func LocalAppData(home string) string { return pick("LOCALAPPDATA", home, "AppData", "Local") }

// HostConfigHome and HostDataHome are the host user's folders, for reading another program's files. Outside a
// Flatpak they equal ConfigHome and DataHome. Inside one (flatpak.Inside) Flatpak points XDG_CONFIG_HOME and
// XDG_DATA_HOME at the app's own ~/.var/app folder, so those are skipped; HOST_XDG_CONFIG_HOME and
// HOST_XDG_DATA_HOME, which Flatpak sets to the host's custom folders, are read under the same rule, else the
// default under home is returned.
func HostConfigHome(home string) string { return hostPick("XDG_CONFIG_HOME", home, ".config") }

// HostDataHome is HostConfigHome for the data home.
func HostDataHome(home string) string { return hostPick("XDG_DATA_HOME", home, ".local", "share") }

// FlatpakApp is home/.var/app/<appID>, the per-user folder Flatpak gives each app.
func FlatpakApp(home, appID string) string {
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".var", "app", appID)
}

func hostPick(name, home string, def ...string) string {
	if flatpak.Inside() {
		return pick("HOST_"+name, home, def...)
	}
	return pick(name, home, def...)
}

func pick(name, home string, def ...string) string {
	if home == "" {
		return ""
	}
	if v := os.Getenv(name); filepath.IsAbs(v) && isRealHome(home) {
		return v
	}
	return filepath.Join(append([]string{home}, def...)...)
}

func isRealHome(home string) bool {
	own, err := os.UserHomeDir()
	return err == nil && foldCase(filepath.Clean(own)) == foldCase(filepath.Clean(home))
}
