package nativemessaging

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-desktopenv/internal/hostfile"
	"github.com/Rethunk-Tech/go-desktopenv/userdirs"
	"github.com/Rethunk-Tech/go-safefs"
)

// browsers are where each browser keeps its profile: under the host's config folder, or for Firefox under home.
// A browser counts as installed when that folder exists.
var browsers = []struct {
	dir, name string
	firefox   bool
}{
	{dir: "google-chrome", name: "Google Chrome"},
	{dir: "google-chrome-beta", name: "Google Chrome Beta"},
	{dir: "google-chrome-unstable", name: "Google Chrome Dev"},
	{dir: "chromium", name: "Chromium"},
	{dir: "BraveSoftware/Brave-Browser", name: "Brave"},
	{dir: "microsoft-edge", name: "Microsoft Edge"},
	{dir: "vivaldi", name: "Vivaldi"},
	{dir: "vivaldi-snapshot", name: "Vivaldi Snapshot"},
	{dir: ".mozilla", name: "Firefox", firefox: true},
}

type location struct {
	browser  string
	firefox  bool
	manifest string
}

func (h Host) host() hostfile.Host {
	if h.Run != nil {
		return hostfile.Host{Run: h.Run}
	}
	return hostfile.Host{Run: hostfile.Exec}
}

// locations are the installed browsers, each with where it looks for this host's manifest.
func (h Host) locations() ([]location, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	config := hostfile.ConfigHome(home)
	var out []location
	for _, b := range browsers {
		dir := filepath.Join(config, b.dir)
		manifest := filepath.Join(dir, "NativeMessagingHosts", h.Name+".json")
		if b.firefox {
			dir = filepath.Join(home, b.dir)
			manifest = filepath.Join(dir, "native-messaging-hosts", h.Name+".json")
		}
		if h.host().IsDir(dir) {
			out = append(out, location{browser: b.name, firefox: b.firefox, manifest: manifest})
		}
	}
	return out, nil
}

// exe is what the manifests run: Path, or inside a Flatpak the wrapper in the app's own folder under ~/.var/app,
// which the sandbox and the host see at the same path.
func (h Host) exe() (string, error) {
	if !flatpak.Inside() {
		return h.Path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userdirs.FlatpakApp(home, os.Getenv("FLATPAK_ID")), "native-host"), nil
}

var shellSafe = regexp.MustCompile(`^[A-Za-z0-9._/=+-]+$`)

func shellQuote(s string) string {
	if shellSafe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Install writes the manifest into every installed browser's folder; a browser installed later needs another call.
// Inside a Flatpak it first writes the wrapper the host browser starts, since the host cannot run the sandbox's
// Path; the browser's arguments pass through to the program.
func (h Host) Install() error {
	exe, err := h.exe()
	if err != nil {
		return err
	}
	if flatpak.Inside() {
		wrapper := "#!/bin/sh\nexec flatpak run " + shellQuote("--command="+filepath.Base(h.Path)) + " " +
			shellQuote(os.Getenv("FLATPAK_ID")) + " \"$@\"\n"
		if err := os.MkdirAll(filepath.Dir(exe), 0o700); err != nil {
			return err
		}
		if err := safefs.WriteFile(exe, []byte(wrapper), 0o700); err != nil {
			return err
		}
	}
	locs, err := h.locations()
	if err != nil {
		return err
	}
	var errs []error
	for _, l := range locs {
		m, err := h.manifest(exe, l.firefox)
		if err == nil {
			err = h.host().Write(l.manifest, m, 0o644)
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// Uninstall removes the manifest from every installed browser's folder.
func (h Host) Uninstall() error {
	locs, err := h.locations()
	if err != nil {
		return err
	}
	var errs []error
	for _, l := range locs {
		errs = append(errs, h.host().Remove(l.manifest))
	}
	return errors.Join(errs...)
}

// Status is each installed browser's manifest, in a fixed browser order.
func (h Host) Status() ([]Status, error) {
	exe, err := h.exe()
	if err != nil {
		return nil, err
	}
	locs, err := h.locations()
	if err != nil {
		return nil, err
	}
	var out []Status
	for _, l := range locs {
		b, err := h.host().Read(l.manifest)
		out = append(out, Status{Browser: l.browser, Manifest: l.manifest, State: stateOf(exe, b, err)})
	}
	return out, nil
}
