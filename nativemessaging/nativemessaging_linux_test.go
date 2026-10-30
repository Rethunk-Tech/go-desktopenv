package nativemessaging

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func testHost(path string) Host {
	return Host{
		Name: "org.example.app", Description: "Example", Path: path,
		AllowedOrigins: []string{"chrome-extension://abc/"}, AllowedExtensions: []string{"app@example.org"},
	}
}

// fakeHome gives the test its own home and config folder, outside any Flatpak, and returns them.
func fakeHome(t *testing.T) (home, config string) {
	t.Helper()
	home = t.TempDir()
	config = filepath.Join(home, "cfg")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("HOST_XDG_CONFIG_HOME", "")
	t.Setenv("FLATPAK_ID", "")
	return home, config
}

func mkdirs(t *testing.T, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
}

type manifest struct {
	Name              string   `json:"name"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`
	AllowedOrigins    []string `json:"allowed_origins"`
	AllowedExtensions []string `json:"allowed_extensions"`
}

func readManifest(t *testing.T, path string) manifest {
	t.Helper()
	var m manifest
	b, err := os.ReadFile(path) //nolint:gosec // G304: a path under the test's own temp folder
	if err != nil || json.Unmarshal(b, &m) != nil {
		t.Fatalf("%s: %s, %v", path, b, err)
	}
	return m
}

func TestInstallWritesInstalledBrowsersOnlyAndUninstallRemovesThem(t *testing.T) {
	home, config := fakeHome(t)
	vivaldi := filepath.Join(config, "vivaldi")
	mkdirs(t, vivaldi, filepath.Join(home, ".mozilla"))
	h := testHost("/opt/app/app")
	if err := h.Install(); err != nil {
		t.Fatal(err)
	}
	chromium := filepath.Join(vivaldi, "NativeMessagingHosts", "org.example.app.json")
	firefox := filepath.Join(home, ".mozilla", "native-messaging-hosts", "org.example.app.json")
	if m := readManifest(t, chromium); m.Path != h.Path || m.Type != "stdio" || m.Name != h.Name ||
		!slices.Equal(m.AllowedOrigins, h.AllowedOrigins) || m.AllowedExtensions != nil {
		t.Errorf("chromium manifest %+v", m)
	}
	if m := readManifest(t, firefox); m.AllowedOrigins != nil || !slices.Equal(m.AllowedExtensions, h.AllowedExtensions) {
		t.Errorf("firefox manifest %+v", m)
	}
	if _, err := os.Stat(filepath.Join(config, "google-chrome")); err == nil {
		t.Error("created a folder for a browser that is not installed")
	}
	if err := h.Uninstall(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{chromium, firefox} {
		if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s survived Uninstall: %v", p, err)
		}
	}
}

func TestStatusOkMissingStaleAndUnreadable(t *testing.T) {
	home, config := fakeHome(t)
	chrome := filepath.Join(config, "google-chrome")
	brave := filepath.Join(config, "BraveSoftware", "Brave-Browser")
	edge := filepath.Join(config, "microsoft-edge")
	mkdirs(t, chrome, filepath.Join(home, ".mozilla"))
	if err := testHost(filepath.Join(home, "old-app")).Install(); err != nil {
		t.Fatal(err)
	}
	h := testHost(filepath.Join(home, "app"))
	m, err := h.Manifest(false)
	if err != nil {
		t.Fatal(err)
	}
	mkdirs(t, filepath.Join(chrome, "NativeMessagingHosts"), brave, filepath.Join(edge, "NativeMessagingHosts"))
	for path, body := range map[string][]byte{
		filepath.Join(chrome, "NativeMessagingHosts", "org.example.app.json"): m,
		filepath.Join(edge, "NativeMessagingHosts", "org.example.app.json"):   []byte(`{"path":""}`),
	} {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := h.Status()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]State{}
	for _, r := range rows {
		got[r.Browser] = r.State
	}
	want := map[string]State{"Google Chrome": OK, "Brave": Missing, "Microsoft Edge": Unreadable, "Firefox": Stale}
	if len(got) != len(want) {
		t.Fatalf("rows %+v", rows)
	}
	for b, s := range want {
		if got[b] != s {
			t.Errorf("%s: %q, want %q", b, got[b], s)
		}
	}
}

// inFlatpak puts a flatpak-spawn on PATH that runs the host command here, so the test's folders are the host's.
func inFlatpak(t *testing.T) {
	t.Helper()
	bin := t.TempDir()
	spawn := filepath.Join(bin, "flatpak-spawn")
	if err := os.WriteFile(spawn, []byte("#!/bin/sh\n[ \"$1\" = --host ] || exit 99\nshift\nexec \"$@\"\n"), 0o700); err != nil { //nolint:gosec // G306: the fake must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FLATPAK_ID", "org.example.App")
}

func TestFlatpakWritesHostManifestsThatStartTheAppThroughFlatpak(t *testing.T) {
	home, sandboxConfig := fakeHome(t)
	inFlatpak(t)
	vivaldi := filepath.Join(home, ".config", "vivaldi")
	mkdirs(t, vivaldi, filepath.Join(home, ".mozilla"))
	h := testHost("/app/bin/my app")
	if err := h.Install(); err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(home, ".var", "app", "org.example.App", "native-host")
	b, err := os.ReadFile(wrapper) //nolint:gosec // G304: a path under the test's own temp folder
	if err != nil || !strings.Contains(string(b), `exec flatpak run '--command=my app' org.example.App "$@"`) {
		t.Fatalf("wrapper %q, %v", b, err)
	}
	chromium := filepath.Join(vivaldi, "NativeMessagingHosts", "org.example.app.json")
	firefox := filepath.Join(home, ".mozilla", "native-messaging-hosts", "org.example.app.json")
	for _, p := range []string{chromium, firefox} {
		if m := readManifest(t, p); m.Path != wrapper {
			t.Errorf("%s runs %q", p, m.Path)
		}
	}
	rows, err := h.Status()
	if err != nil || len(rows) != 2 {
		t.Fatalf("status %+v, %v", rows, err)
	}
	for _, s := range rows {
		if s.State != OK {
			t.Errorf("status %+v", s)
		}
	}
	if _, err := os.Stat(filepath.Join(sandboxConfig, "vivaldi")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("wrote into the sandbox's config: %v", err)
	}
	if err := h.Uninstall(); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{chromium, firefox} {
		if _, err := os.Stat(p); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s survived Uninstall: %v", p, err)
		}
	}
}
