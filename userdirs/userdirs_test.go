package userdirs

import (
	"path/filepath"
	"testing"
)

// realHome points the process's own home at a temp folder, the only home the environment is allowed to describe.
func realHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, k := range []string{"XDG_CONFIG_HOME", "XDG_DATA_HOME", "APPDATA", "LOCALAPPDATA", "HOST_XDG_CONFIG_HOME", "HOST_XDG_DATA_HOME", "FLATPAK_ID"} {
		t.Setenv(k, "")
	}
	return home
}

type fn struct {
	name string
	env  string
	call func(string) string
	def  []string
}

var own = []fn{
	{"ConfigHome", "XDG_CONFIG_HOME", ConfigHome, []string{".config"}},
	{"DataHome", "XDG_DATA_HOME", DataHome, []string{".local", "share"}},
	{"AppData", "APPDATA", AppData, []string{"AppData", "Roaming"}},
	{"LocalAppData", "LOCALAPPDATA", LocalAppData, []string{"AppData", "Local"}},
}

func TestOwnFolders(t *testing.T) {
	for _, f := range own {
		t.Run(f.name, func(t *testing.T) {
			home := realHome(t)
			def := filepath.Join(append([]string{home}, f.def...)...)
			if got := f.call(home); got != def {
				t.Errorf("unset: %q, want %q", got, def)
			}
			abs := filepath.Join(t.TempDir(), "custom")
			t.Setenv(f.env, abs)
			if got := f.call(home); got != abs {
				t.Errorf("absolute: %q, want %q", got, abs)
			}
			if got := f.call(filepath.Join(home, "..", filepath.Base(home))); got != abs {
				t.Errorf("unclean spelling of the real home: %q, want %q", got, abs)
			}
			other := t.TempDir()
			if got, want := f.call(other), filepath.Join(append([]string{other}, f.def...)...); got != want {
				t.Errorf("another home: %q, want %q", got, want)
			}
			t.Setenv(f.env, "relative/dir")
			if got := f.call(home); got != def {
				t.Errorf("relative: %q, want %q", got, def)
			}
			t.Setenv(f.env, "")
			if got := f.call(home); got != def {
				t.Errorf("empty: %q, want %q", got, def)
			}
			if got := f.call(""); got != "" {
				t.Errorf("empty home: %q", got)
			}
		})
	}
}

func TestHostFolders(t *testing.T) {
	for _, f := range []fn{
		{"HostConfigHome", "XDG_CONFIG_HOME", HostConfigHome, []string{".config"}},
		{"HostDataHome", "XDG_DATA_HOME", HostDataHome, []string{".local", "share"}},
	} {
		t.Run(f.name, func(t *testing.T) {
			home := realHome(t)
			def := filepath.Join(append([]string{home}, f.def...)...)
			custom := filepath.Join(t.TempDir(), "custom")

			t.Setenv(f.env, custom)
			if got := f.call(home); got != custom {
				t.Errorf("outside a Flatpak follows %s: %q, want %q", f.env, got, custom)
			}
			t.Setenv("HOST_"+f.env, filepath.Join(t.TempDir(), "host"))
			if got := f.call(home); got != custom {
				t.Errorf("HOST_ variable outside a Flatpak is ignored: %q", got)
			}

			t.Setenv("FLATPAK_ID", "org.example.App")
			t.Setenv("HOST_"+f.env, "")
			if got := f.call(home); got != def {
				t.Errorf("inside, the app's own %s is skipped: %q, want %q", f.env, got, def)
			}
			host := filepath.Join(t.TempDir(), "host")
			t.Setenv("HOST_"+f.env, host)
			if got := f.call(home); got != host {
				t.Errorf("inside, HOST_%s: %q, want %q", f.env, got, host)
			}
			other := t.TempDir()
			if got, want := f.call(other), filepath.Join(append([]string{other}, f.def...)...); got != want {
				t.Errorf("inside, another home: %q, want %q", got, want)
			}
			t.Setenv("HOST_"+f.env, "relative")
			if got := f.call(home); got != def {
				t.Errorf("inside, relative HOST_%s: %q, want %q", f.env, got, def)
			}
			if got := f.call(""); got != "" {
				t.Errorf("empty home: %q", got)
			}
		})
	}
}

func TestFlatpakApp(t *testing.T) {
	t.Parallel()
	home := filepath.Join("h", "u")
	if got, want := FlatpakApp(home, "com.valvesoftware.Steam"), filepath.Join(home, ".var", "app", "com.valvesoftware.Steam"); got != want {
		t.Errorf("%q, want %q", got, want)
	}
	if got := FlatpakApp("", "x"); got != "" {
		t.Errorf("empty home: %q", got)
	}
}
