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

func TestEnableRejectsAnIDThatIsNotOnePathElement(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("FLATPAK_ID", "")
	for _, id := range []string{"", ".", "..", "a/b", `a\b`, "../evil"} {
		if err := Enable(t.Context(), Entry{ID: id, Argv: []string{"/bin/true"}}); err == nil {
			t.Errorf("Enable(%q): no error", id)
		}
		if err := Disable(t.Context(), id); err == nil {
			t.Errorf("Disable(%q): no error", id)
		}
		if _, err := Enabled(id); err == nil {
			t.Errorf("Enabled(%q): no error", id)
		}
	}
	entries, err := os.ReadDir(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("wrote under config home for a rejected id: %v", entries)
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

func TestEnableShowsTheIDWhenNameIsEmpty(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("FLATPAK_ID", "")
	argv := []string{"/bin/true"}
	e := Entry{ID: "myapp", Argv: argv}
	if err := Enable(t.Context(), e); err != nil {
		t.Fatal(err)
	}
	if err := Enable(t.Context(), e); err != nil {
		t.Fatal(err)
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
	want := "[Desktop Entry]\nType=Application\nName=myapp\nExec=" + desktopentry.Exec(argv...) + "\nX-GNOME-Autostart-enabled=true\n"
	if string(body) != want {
		t.Fatalf("desktop file = %q, want %q", body, want)
	}
	if err := Disable(t.Context(), "myapp"); err != nil {
		t.Fatal(err)
	}
	if err := Disable(t.Context(), "myapp"); err != nil {
		t.Fatal(err)
	}
}

func TestEnableRejectsAnEmptyCommand(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("FLATPAK_ID", "")
	if err := Enable(t.Context(), Entry{ID: "myapp"}); err == nil {
		t.Fatal("empty Argv: no error")
	}
	if _, err := os.Stat(filepath.Join(cfg, "autostart")); !os.IsNotExist(err) {
		t.Fatalf("wrote autostart for an empty command: %v", err)
	}
}

func TestEnabledIsFalseWhenConfigHomeIsMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "no-such"))
	t.Setenv("FLATPAK_ID", "")
	on, err := Enabled("myapp")
	if err != nil || on {
		t.Fatalf("Enabled = %v, %v", on, err)
	}
}

func TestEnableInsideAFlatpakDoesNotWriteTheSandboxAutostartFile(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("FLATPAK_ID", "org.example.App")
	// A missing session bus is still the portal path; the sandbox must not grow an entry.
	_ = Enable(t.Context(), Entry{ID: "myapp", Argv: []string{"/bin/true"}})
	entries, err := os.ReadDir(cfg)
	if err != nil || len(entries) != 0 {
		t.Fatalf("sandbox config = %v, %v", entries, err)
	}
}
