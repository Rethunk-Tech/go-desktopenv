package schemehandler

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-desktopenv/desktopentry"
	"github.com/Rethunk-Tech/go-desktopenv/internal/hostfile"
)

const appID = "org.example.App.desktop"

// recorder plays xdg-mime: query answers the current default, default changes it. Nothing touches the real system.
type recorder struct {
	calls  []string
	byMime map[string]string
}

func (r *recorder) run(name string, args ...string) (string, error) {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	if name == "flatpak-spawn" && len(args) > 2 && args[0] == "--host" && args[1] == "xdg-mime" {
		name, args = xdgMime, args[2:]
	}
	if name != xdgMime {
		return "", nil
	}
	switch args[0] {
	case "query":
		return r.byMime[args[len(args)-1]] + "\n", nil
	case "default":
		r.byMime[args[2]] = args[1]
	}
	return "", nil
}

func testConfig() Config {
	cfg := Config{Exe: "/opt/app/app", Schemes: []string{"nxm", "ror2mm"}, DesktopID: appID}
	cfg.Entry.Name, cfg.Entry.Comment, cfg.Entry.Icon = "App", "An app", "app"
	cfg.Entry.Categories = []string{"Game", "Utility"}
	cfg.Entry.Keywords = []string{"mod"}
	cfg.Entry.StartupWMClass = "org.example.App"
	cfg.Entry.MimeTypes = []string{"x-scheme-handler/example", "application/x-example"}
	return cfg
}

// newLinux gives the test its own home and XDG folders under one temp folder (data, config, system) and a recorder
// in place of xdg-mime.
func newLinux(t *testing.T, cfg Config, nxmOwner string) (*Handler, *recorder) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", filepath.Join(dir, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_DATA_DIRS", filepath.Join(dir, "system"))
	t.Setenv("FLATPAK_ID", "")
	h, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	r := &recorder{byMime: map[string]string{schemeMime("nxm"): nxmOwner}}
	h.run = r.run
	return h, r
}

// under is a folder newLinux made beside the data home.
func under(h *Handler, name string) string { return filepath.Join(filepath.Dir(h.dataHome), name) }

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // G304: a path under the test's own temp folder
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	return string(b)
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestRegisterThenRestorePreviousHandler(t *testing.T) {
	h, r := newLinux(t, testConfig(), "vortex.desktop")
	owner, err := h.Owner("nxm")
	if err != nil || owner.ID != "vortex.desktop" || owner.Mine || owner.Name != "vortex" {
		t.Fatalf("owner before: %+v, %v", owner, err)
	}
	if err := h.Register("nxm", "ror2mm"); err != nil {
		t.Fatal(err)
	}
	if r.byMime[schemeMime("nxm")] != appID || r.byMime[schemeMime("ror2mm")] != appID {
		t.Fatalf("defaults after Register: %v", r.byMime)
	}
	want := `[Desktop Entry]
Type=Application
Name=App
Comment=An app
Exec="/opt/app/app" %u
Icon=app
Terminal=false
Categories=Game;Utility;
Keywords=mod;
StartupWMClass=org.example.App
MimeType=x-scheme-handler/nxm;x-scheme-handler/ror2mm;x-scheme-handler/example;application/x-example;
`
	if got := read(t, h.desktopPath()); got != want {
		t.Fatalf("desktop file:\n%s", got)
	}
	if owner, _ := h.Owner("nxm"); !owner.Mine || owner.Name != "App" {
		t.Fatalf("owner after Register: %+v", owner)
	}
	if err := h.Restore(map[string]string{"nxm": "vortex.desktop", "ror2mm": "r2modman.desktop"}); err != nil {
		t.Fatal(err)
	}
	if r.byMime[schemeMime("nxm")] != "vortex.desktop" || r.byMime[schemeMime("ror2mm")] != "r2modman.desktop" {
		t.Errorf("defaults after Restore: %v", r.byMime)
	}
	if got := read(t, h.desktopPath()); strings.Contains(got, schemeMime("nxm")) || !strings.Contains(got, "MimeType=x-scheme-handler/example;application/x-example;") {
		t.Errorf("desktop file after Restore: %s", got)
	}
}

func TestRestoreWithoutPreviousDropsOurDefaultThroughASymlink(t *testing.T) {
	h, r := newLinux(t, testConfig(), "")
	if err := h.Register("nxm"); err != nil {
		t.Fatal(err)
	}
	config := under(h, "config")
	target := filepath.Join(config, "dotfiles.list")
	write(t, target, "[Default Applications]\nx-scheme-handler/nxm="+appID+";\ntext/html=a.desktop\n")
	list := filepath.Join(config, "mimeapps.list")
	if err := os.Symlink(target, list); err != nil {
		t.Fatal(err)
	}
	r.calls = nil
	if err := h.Restore(nil); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(list); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("mimeapps.list symlink replaced: %v", err)
	}
	if got := read(t, target); got != "[Default Applications]\ntext/html=a.desktop\n" {
		t.Errorf("mimeapps.list: %q", got)
	}
	for _, c := range r.calls {
		if strings.HasPrefix(c, xdgMime+" default") {
			t.Errorf("Restore set a default: %s", c)
		}
	}
}

func TestReleaseHandsBackOnlyTheGivenScheme(t *testing.T) {
	h, r := newLinux(t, testConfig(), "vortex.desktop")
	if err := h.Release([]string{"ror2mm"}, map[string]string{"ror2mm": "r2modman.desktop"}); err != nil {
		t.Fatal(err)
	}
	if r.byMime[schemeMime("ror2mm")] != "r2modman.desktop" || r.byMime[schemeMime("nxm")] != "vortex.desktop" {
		t.Fatalf("defaults: %v", r.byMime)
	}
}

func TestWriteEntryKeepsTheSchemesAndIsIdempotent(t *testing.T) {
	h, r := newLinux(t, testConfig(), "")
	if err := h.WriteEntry(); err != nil {
		t.Fatal(err)
	}
	if got := read(t, h.desktopPath()); strings.Contains(got, schemeMime("nxm")) || !strings.Contains(got, "MimeType=x-scheme-handler/example;") {
		t.Fatalf("first entry lists the schemes: %s", got)
	}
	r.calls = nil
	if err := h.WriteEntry(); err != nil || len(r.calls) != 0 {
		t.Fatalf("a second run rewrote the entry: %q, %v", r.calls, err)
	}
	if err := h.Register("nxm"); err != nil {
		t.Fatal(err)
	}
	if err := h.WriteEntry(); err != nil {
		t.Fatal(err)
	}
	if got := read(t, h.desktopPath()); !strings.Contains(got, schemeMime("nxm")) {
		t.Errorf("WriteEntry dropped the schemes: %s", got)
	}
}

func TestEntryProgramReadsBackWhateverThePathHolds(t *testing.T) {
	cfg := testConfig()
	cfg.Exe = "/opt/50%off/$x `b` \\q \"z\" d"
	h, _ := newLinux(t, cfg, "")
	if _, err := h.EntryProgram(); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("no entry: %v", err)
	}
	if err := h.WriteEntry(); err != nil {
		t.Fatal(err)
	}
	if got, err := h.EntryProgram(); err != nil || got != cfg.Exe {
		t.Fatalf("EntryProgram = %q, %v", got, err)
	}
	if err := h.RemoveEntry(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(h.desktopPath()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("entry survived RemoveEntry: %v", err)
	}
}

func TestPackagedSetsDefaultsWithoutWriting(t *testing.T) {
	cfg := testConfig()
	cfg.Packaged = true
	h, r := newLinux(t, cfg, "vortex.desktop")
	// The package's system entry is the handler even beside an integrator's for the same program.
	writeApp(t, h, gearLever, gearLeverEntry(cfg.Exe))
	if err := h.Register("nxm"); err != nil {
		t.Fatal(err)
	}
	if err := h.WriteEntry(); err != nil {
		t.Fatal(err)
	}
	if r.byMime[schemeMime("nxm")] != appID {
		t.Fatalf("default is %q after packaged Register", r.byMime[schemeMime("nxm")])
	}
	if _, err := os.Stat(h.desktopPath()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatal("packaged Register wrote a user desktop entry")
	}
}

func writeApp(t *testing.T, h *Handler, name, body string) {
	t.Helper()
	write(t, filepath.Join(h.apps(), name), body)
}

const gearLever = "app.appimage_0.desktop"

func gearLeverEntry(exe string) string {
	return "[Desktop Entry]\nName=App\nExec=env DESKTOPINTEGRATION=1 " + exe + " %u\nX-AppImage-Name=App\n"
}

func TestNoIntegratorWritesOurEntry(t *testing.T) {
	h, r := newLinux(t, testConfig(), "")
	// Our own shortcut and an entry that cannot take a URL do not count as an integrator's.
	writeApp(t, h, "org.example.App.play-x.desktop", "[Desktop Entry]\nExec=\"/opt/app/app\" --play=x %u\n")
	writeApp(t, h, "other.desktop", "[Desktop Entry]\nExec=/opt/app/app\n")
	if err := h.Register("nxm"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(h.desktopPath()); err != nil {
		t.Fatalf("own entry: %v", err)
	}
	if r.byMime[schemeMime("nxm")] != appID {
		t.Fatalf("defaults: %v", r.byMime)
	}
}

func TestAnIntegratorsEntryIsTheHandler(t *testing.T) {
	cfg := testConfig()
	cfg.Exe = "/home/u/Apps/app.appimage_0.appimage"
	h, r := newLinux(t, cfg, "vortex.desktop")
	writeApp(t, h, "appimagekit_other.desktop", "[Desktop Entry]\nExec=\""+cfg.Exe+"\" %U\n")
	writeApp(t, h, gearLever, gearLeverEntry(cfg.Exe))
	for range 2 {
		if err := h.Register("nxm"); err != nil {
			t.Fatal(err)
		}
		if err := h.WriteEntry(); err != nil {
			t.Fatal(err)
		}
		if err := h.SetDefault(cfg.Entry.MimeTypes...); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := os.Stat(h.desktopPath()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("wrote its own entry beside the integrator's: %v", err)
	}
	for _, mime := range append([]string{schemeMime("nxm")}, cfg.Entry.MimeTypes...) {
		if r.byMime[mime] != gearLever {
			t.Errorf("%s default %q", mime, r.byMime[mime])
		}
	}
	if owner, _ := h.Owner("nxm"); !owner.Mine || owner.Name != "App" {
		t.Errorf("owner: %+v", owner)
	}
}

func TestAnIntegratorReplacesOurStaleEntry(t *testing.T) {
	cfg := testConfig()
	cfg.Exe = "/home/u/Apps/app.appimage_0.appimage"
	h, r := newLinux(t, cfg, "")
	if err := h.Register(cfg.Schemes...); err != nil {
		t.Fatal(err)
	}
	if err := h.SetDefault(cfg.Entry.MimeTypes...); err != nil {
		t.Fatal(err)
	}
	r.byMime["x-scheme-handler/http"] = "firefox.desktop"
	writeApp(t, h, gearLever, gearLeverEntry(cfg.Exe))
	r.calls = nil
	if err := h.WriteEntry(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(h.desktopPath()); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("stale own entry kept: %v", err)
	}
	if !slices.Contains(r.calls, updateDB+" "+h.apps()) {
		t.Errorf("desktop database not rebuilt: %q", r.calls)
	}
	for _, mime := range append(h.schemeMimes(), cfg.Entry.MimeTypes...) {
		if r.byMime[mime] != gearLever {
			t.Errorf("%s default %q", mime, r.byMime[mime])
		}
	}
	if r.byMime["x-scheme-handler/http"] != "firefox.desktop" {
		t.Error("another type's default moved")
	}
}

// inFlatpak is newLinux inside a Flatpak whose host is this machine: a host sh script runs here, against the test's
// folders, xdg-mime goes to the recorder, and a host gio only records what it was asked to launch.
func inFlatpak(t *testing.T, nxmOwner string) (*Handler, *recorder, string) {
	t.Helper()
	h, r := newLinux(t, testConfig(), nxmOwner)
	t.Setenv("FLATPAK_ID", "org.example.App")
	bin := t.TempDir()
	launched := filepath.Join(bin, "launched")
	write(t, filepath.Join(bin, "gio"), "#!/bin/sh\nprintf '%s\\n' \"$@\" > "+launched+"\n")
	if err := os.Chmod(filepath.Join(bin, "gio"), 0o700); err != nil { //nolint:gosec // G302: the fake gio must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	h.run = func(name string, args ...string) (string, error) {
		if name != "flatpak-spawn" || len(args) < 2 || args[0] != "--host" {
			t.Fatalf("ran %s %q outside the host bridge", name, args)
		}
		if args[1] == "xdg-mime" {
			return r.run(name, args...)
		}
		if args[1] != "sh" {
			t.Fatalf("host command %q: the bridge only runs sh scripts and xdg-mime", args[1])
		}
		return hostfile.Exec("sh", args[2:]...)
	}
	return h, r, launched
}

func TestFlatpakRestoreDropsOurDefaultFromTheHostsMimeapps(t *testing.T) {
	h, r, _ := inFlatpak(t, "")
	hostConfig := filepath.Join(under(h, "home"), ".config")
	list := filepath.Join(hostConfig, "mimeapps.list")
	write(t, list, "[Default Applications]\nx-scheme-handler/nxm="+appID+";\ntext/html=a.desktop\n")
	if err := h.Register("nxm"); err != nil {
		t.Fatal(err)
	}
	if owner, err := h.Owner("nxm"); err != nil || !owner.Mine {
		t.Fatalf("owner = %+v, %v", owner, err)
	}
	if err := h.Restore(nil); err != nil {
		t.Fatal(err)
	}
	if got := read(t, list); got != "[Default Applications]\ntext/html=a.desktop\n" {
		t.Errorf("host mimeapps.list: %q", got)
	}
	if _, err := os.Stat(filepath.Join(under(h, "config"), "mimeapps.list")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("wrote the sandbox's mimeapps.list: %v", err)
	}
	if !slices.Contains(r.calls, "flatpak-spawn --host xdg-mime default "+appID+" "+schemeMime("nxm")) {
		t.Errorf("calls %q", r.calls)
	}
}

func TestFlatpakForwardsToThePreviousHandlerOnTheHost(t *testing.T) {
	h, _, launched := inFlatpak(t, "")
	entry := filepath.Join(h.apps(), "other.desktop")
	write(t, entry, "[Desktop Entry]\nExec=other %u\n")
	if err := h.Forward("nxm://stardewvalley/mods/1", "other.desktop"); err != nil {
		t.Fatal(err)
	}
	if got := read(t, launched); got != "launch\n"+entry+"\nnxm://stardewvalley/mods/1\n" {
		t.Fatalf("host gio got %q", got)
	}
	if err := h.Forward("nxm://x", "missing.desktop"); err == nil {
		t.Fatal("a missing entry forwarded")
	}
}

func TestForwardRunsThePreviousEntrysExec(t *testing.T) {
	h, _ := newLinux(t, testConfig(), "")
	out := filepath.Join(t.TempDir(), "out")
	write(t, filepath.Join(under(h, "system"), "applications", "other.desktop"),
		"[Desktop Entry]\nName=Other\nExec="+desktopentry.Exec("sh", "-c", `printf %s "$1" > "$0"`, out)+" %u\n")
	h.run = hostfile.Exec
	link := "nxm://x/mods/1?a=$(id)&b"
	if err := h.Forward(link, "other.desktop"); err != nil {
		t.Fatal(err)
	}
	if got := read(t, out); got != link {
		t.Fatalf("handler got %q", got)
	}
	var exit *exec.ExitError
	if err := h.Forward(link, ""); err == nil || errors.As(err, &exit) {
		t.Fatalf("no previous handler: %v", err)
	}
}
