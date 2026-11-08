//go:build windows

package autostart

import (
	"os"
	"strconv"
	"testing"

	winreg "golang.org/x/sys/windows/registry"
)

func TestEnableSetsRunValue(t *testing.T) {
	app := "DesktopEnvAutostartTest-" + strconv.Itoa(os.Getpid())
	t.Cleanup(func() { _ = Disable(t.Context(), app) })
	on, err := Enabled(app)
	if err != nil || on {
		t.Fatalf("Enabled before Enable = %v, %v", on, err)
	}
	exe := `C:\Program Files\App\app.exe`
	if err := Enable(t.Context(), Entry{ID: app, Argv: []string{exe}}); err != nil {
		t.Fatal(err)
	}
	on, err = Enabled(app)
	if err != nil || !on {
		t.Fatalf("Enabled = %v, %v", on, err)
	}
	key, err := winreg.OpenKey(winreg.CURRENT_USER, runKey, winreg.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = key.Close() }()
	got, _, err := key.GetStringValue(app)
	if err != nil {
		t.Fatal(err)
	}
	want := `"` + exe + `"`
	if got != want {
		t.Fatalf("Run value = %q, want %q", got, want)
	}
	if err := Disable(t.Context(), app); err != nil {
		t.Fatal(err)
	}
	on, err = Enabled(app)
	if err != nil || on {
		t.Fatalf("Enabled after Disable = %v, %v", on, err)
	}
}

func TestDisableMissingRunValue(t *testing.T) {
	if err := Disable(t.Context(), "DesktopEnvAutostartTestMissing"); err != nil {
		t.Fatal(err)
	}
}
