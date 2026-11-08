//go:build unix

package autostart

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Rethunk-Tech/go-desktopenv/desktopentry"
)

func TestEnableWritesAutostartFile(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("FLATPAK_ID", "")
	argv := []string{"/usr/bin/my app", "--flag"}
	if err := Enable(t.Context(), Entry{ID: "myapp", Name: "My App", Argv: argv}); err != nil {
		t.Fatal(err)
	}
	on, err := Enabled("myapp")
	if err != nil || !on {
		t.Fatalf("Enabled = %v, %v", on, err)
	}
	root, err := os.OpenRoot(filepath.Join(cfg, "autostart"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	body, err := root.ReadFile("myapp.desktop")
	if err != nil {
		t.Fatal(err)
	}
	want := "[Desktop Entry]\nType=Application\nName=My App\nExec=" + desktopentry.Exec(argv...) + "\nX-GNOME-Autostart-enabled=true\n"
	if string(body) != want {
		t.Fatalf("desktop file = %q, want %q", body, want)
	}
	if err := Disable(t.Context(), "myapp"); err != nil {
		t.Fatal(err)
	}
	if _, err := root.Stat("myapp.desktop"); !os.IsNotExist(err) {
		t.Fatalf("expected desktop file gone, stat = %v", err)
	}
	on, err = Enabled("myapp")
	if err != nil || on {
		t.Fatalf("Enabled after Disable = %v, %v", on, err)
	}
}

func TestDisableMissingAutostartDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FLATPAK_ID", "")
	if err := Disable(t.Context(), "myapp"); err != nil {
		t.Fatal(err)
	}
	on, err := Enabled("myapp")
	if err != nil || on {
		t.Fatalf("Enabled = %v, %v", on, err)
	}
}
