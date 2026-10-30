package nativemessaging

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/go-safefs"
	"golang.org/x/sys/windows/registry"
)

// browsers are the HKCU keys under Software each browser reads: probe exists when it is installed, and its
// NativeMessagingHosts subkey names the manifest. Brave and Vivaldi fall back to the Chromium and Chrome keys.
var browsers = []struct {
	name, probe string
	firefox     bool
}{
	{name: "Google Chrome", probe: `Google\Chrome`},
	{name: "Microsoft Edge", probe: `Microsoft\Edge`},
	{name: "Chromium", probe: `Chromium`},
	{name: "Firefox", probe: `Mozilla`, firefox: true},
}

// software is the HKCU key the browsers' keys live under; tests pass their own.
const software = "Software"

func (h Host) hostKey(root, probe string) string {
	return root + `\` + probe + `\NativeMessagingHosts\` + h.Name
}

func (h Host) manifestPath(firefox bool) (string, error) {
	if h.ManifestDir == "" {
		return "", errors.New("nativemessaging: Host.ManifestDir is required on Windows")
	}
	if firefox {
		return filepath.Join(h.ManifestDir, "firefox.json"), nil
	}
	return filepath.Join(h.ManifestDir, "chromium.json"), nil
}

// Install writes the two manifests into ManifestDir and points every browser's key at its one, installed or not, so
// a browser installed later finds the host too.
func (h Host) Install() error { return h.install(software) }

func (h Host) install(root string) error {
	if _, err := h.manifestPath(false); err != nil {
		return err
	}
	if err := os.MkdirAll(h.ManifestDir, 0o700); err != nil {
		return err
	}
	for _, firefox := range []bool{false, true} {
		path, err := h.manifestPath(firefox)
		if err != nil {
			return err
		}
		m, err := h.Manifest(firefox)
		if err != nil {
			return err
		}
		if err := safefs.WriteFile(path, m, 0o600); err != nil {
			return err
		}
	}
	var errs []error
	for _, b := range browsers {
		path, _ := h.manifestPath(b.firefox)
		errs = append(errs, setDefault(h.hostKey(root, b.probe), path))
	}
	return errors.Join(errs...)
}

func setDefault(key, value string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, key, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	return k.SetStringValue("", value)
}

// Uninstall deletes every browser's key and the two manifests.
func (h Host) Uninstall() error { return h.uninstall(software) }

func (h Host) uninstall(root string) error {
	var errs []error
	for _, b := range browsers {
		if err := registry.DeleteKey(registry.CURRENT_USER, h.hostKey(root, b.probe)); err != nil && !errors.Is(err, registry.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	for _, firefox := range []bool{false, true} {
		path, err := h.manifestPath(firefox)
		if err != nil {
			return errors.Join(append(errs, err)...)
		}
		if err := safefs.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Status is each installed browser's manifest, the one its key names.
func (h Host) Status() ([]Status, error) { return h.status(software), nil }

func (h Host) status(root string) []Status {
	var out []Status
	for _, b := range browsers {
		k, err := registry.OpenKey(registry.CURRENT_USER, root+`\`+b.probe, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		_ = k.Close()
		path, state := h.keyState(h.hostKey(root, b.probe))
		out = append(out, Status{Browser: b.name, Manifest: path, State: state})
	}
	return out
}

func (h Host) keyState(key string) (string, State) {
	k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return key, Missing
	}
	if err != nil {
		return key, Unreadable
	}
	path, _, err := k.GetStringValue("")
	_ = k.Close()
	if err != nil {
		return key, Unreadable
	}
	b, err := safefs.ReadFile(path)
	return path, stateOf(h.Path, b, err)
}
