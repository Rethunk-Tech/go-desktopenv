package nativemessaging

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	"golang.org/x/sys/windows/registry"
)

func deleteTree(t *testing.T, path string) {
	t.Helper()
	k, err := registry.OpenKey(registry.CURRENT_USER, path, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return
	}
	subs, _ := k.ReadSubKeyNames(-1)
	_ = k.Close()
	for _, s := range subs {
		deleteTree(t, path+`\`+s)
	}
	if err := registry.DeleteKey(registry.CURRENT_USER, path); err != nil {
		t.Errorf("delete %s: %v", path, err)
	}
}

func TestInstallPointsEveryBrowserAtItsManifestAndUninstallUndoesIt(t *testing.T) {
	root := `Software\DesktopEnvTest` + strconv.FormatInt(time.Now().UnixNano(), 36)
	t.Cleanup(func() { deleteTree(t, root) })
	h := Host{
		Name: "org.example.app", Path: `C:\App\app.exe`, AllowedOrigins: []string{"chrome-extension://abc/"},
		AllowedExtensions: []string{"app@example.org"}, ManifestDir: t.TempDir(),
	}
	if err := h.install(root); err != nil {
		t.Fatal(err)
	}
	for _, b := range browsers {
		k, err := registry.OpenKey(registry.CURRENT_USER, h.hostKey(root, b.probe), registry.QUERY_VALUE)
		if err != nil {
			t.Fatalf("%s: %v", b.name, err)
		}
		path, _, err := k.GetStringValue("")
		_ = k.Close()
		if err != nil {
			t.Fatal(err)
		}
		var m struct {
			Path              string   `json:"path"`
			AllowedOrigins    []string `json:"allowed_origins"`
			AllowedExtensions []string `json:"allowed_extensions"`
		}
		body, err := os.ReadFile(path) //nolint:gosec // G304: the manifest the test just wrote
		if err != nil || json.Unmarshal(body, &m) != nil || m.Path != h.Path {
			t.Fatalf("%s -> %s: %s, %v", b.name, path, body, err)
		}
		if b.firefox != (len(m.AllowedExtensions) == 1 && m.AllowedOrigins == nil) {
			t.Errorf("%s has the wrong allow-list: %s", b.name, body)
		}
	}
	// Installing created every probe key, so every browser now reads as installed.
	for _, s := range h.status(root) {
		if s.State != OK {
			t.Errorf("status %+v", s)
		}
	}
	if err := h.uninstall(root); err != nil {
		t.Fatal(err)
	}
	for _, s := range h.status(root) {
		if s.State != Missing {
			t.Errorf("after Uninstall: %+v", s)
		}
	}
	if entries, _ := os.ReadDir(h.ManifestDir); len(entries) != 0 {
		t.Errorf("manifests survived Uninstall in %s", h.ManifestDir)
	}
}
