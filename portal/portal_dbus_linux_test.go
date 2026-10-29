package portal

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

const busMarker = "GO_DESKTOPENV_TEST_BUS"

// noBus is set when dbus-run-session is missing, so the bus tests skip (or fail in CI).
var noBus bool

// TestMain runs the tests under a private session bus, so the fake portal never meets a real one. dbus-daemon is a
// real dependency; only the portal answering on it is faked.
func TestMain(m *testing.M) {
	// The live check needs the real session bus, not a private one.
	if os.Getenv(busMarker) == "" && os.Getenv("DESKTOPENV_LIVE_PORTAL") != "1" {
		if run, err := exec.LookPath("dbus-run-session"); err == nil {
			cmd := exec.CommandContext(context.Background(), run, append([]string{"--", os.Args[0]}, os.Args[1:]...)...) //nolint:gosec // G204: re-runs this test binary
			cmd.Env = append(os.Environ(), busMarker+"=1")
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			if err := cmd.Run(); err != nil {
				if exit, ok := errors.AsType[*exec.ExitError](err); ok {
					os.Exit(exit.ExitCode())
				}
				panic(err)
			}
			return
		}
		noBus = true
	}
	os.Exit(m.Run())
}

func requireBus(t *testing.T) {
	t.Helper()
	if noBus || os.Getenv(busMarker) != "1" {
		if os.Getenv("CI") != "" {
			t.Fatal("dbus-run-session is required in CI: install dbus")
		}
		t.Skip("dbus-run-session not on PATH")
	}
}

// fake is a portal.Desktop on the private bus. Its fields choose what the next calls do.
type fake struct {
	mu sync.Mutex

	conn     *dbus.Conn
	response uint32 // Response code sent after a request method
	silent   bool   // never send a Response
	results  map[string]dbus.Variant
	secret   string
	missing  bool         // GetDesktopEntry reports NOENT
	setting  dbus.Variant // Settings.Read answer; notFound overrides it
	notFound bool
	entry    string // Install's entry, recorded
	closed   chan dbus.ObjectPath
	seen     map[string]map[string]dbus.Variant // request method -> options
}

type (
	background fake
	launcher   fake
	secretIf   fake
	settings   fake
	requestObj struct {
		f      *fake
		handle dbus.ObjectPath
	}
)

func startFake(t *testing.T) *fake {
	t.Helper()
	requireBus(t)
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	f := &fake{conn: conn, closed: make(chan dbus.ObjectPath, 4), seen: map[string]map[string]dbus.Variant{}}
	for iface, obj := range map[string]any{
		"org.freedesktop.portal.Background":      (*background)(f),
		"org.freedesktop.portal.DynamicLauncher": (*launcher)(f),
		"org.freedesktop.portal.Secret":          (*secretIf)(f),
		"org.freedesktop.portal.Settings":        (*settings)(f),
	} {
		if err := conn.Export(obj, objectPath, iface); err != nil {
			t.Fatal(err)
		}
	}
	if reply, err := conn.RequestName(busName, 0); err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("RequestName: %v %v", reply, err)
	}
	return f
}

// begin exports the Request object for a call and, unless silent, sends its Response shortly after the reply.
func (f *fake) begin(sender dbus.Sender, method string, options map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	token, _ := options["handle_token"].Value().(string)
	handle := handlePath(string(sender), token)
	f.mu.Lock()
	f.seen[method] = options
	code, silent, results := f.response, f.silent, f.results
	f.mu.Unlock()
	if err := f.conn.Export(&requestObj{f, handle}, handle, requestIf); err != nil {
		return "", dbus.MakeFailedError(err)
	}
	if !silent {
		go func() {
			time.Sleep(20 * time.Millisecond)
			_ = f.conn.Emit(handle, requestIf+".Response", code, results)
		}()
	}
	return handle, nil
}

func (r *requestObj) Close() *dbus.Error {
	r.f.closed <- r.handle
	return nil
}

func (b *background) RequestBackground(s dbus.Sender, _ string, options map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	return (*fake)(b).begin(s, "RequestBackground", options)
}

func (l *launcher) PrepareInstall(s dbus.Sender, _, _ string, _ dbus.Variant, options map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	return (*fake)(l).begin(s, "PrepareInstall", options)
}

func (l *launcher) Install(_, _, entry string, _ map[string]dbus.Variant) *dbus.Error {
	(*fake)(l).mu.Lock()
	defer (*fake)(l).mu.Unlock()
	(*fake)(l).entry = entry
	return nil
}

func (l *launcher) GetDesktopEntry(string) (string, *dbus.Error) {
	(*fake)(l).mu.Lock()
	defer (*fake)(l).mu.Unlock()
	if (*fake)(l).missing {
		return "", &dbus.Error{Name: "org.gtk.GDBus.UnmappedGError.Quark._g_2dfile_2derror_2dquark.Code4"}
	}
	return "[Desktop Entry]\n", nil
}

func (l *launcher) Uninstall(string, map[string]dbus.Variant) *dbus.Error { return nil }

func (s *secretIf) RetrieveSecret(sender dbus.Sender, fd dbus.UnixFD, options map[string]dbus.Variant) (dbus.ObjectPath, *dbus.Error) {
	f := (*fake)(s)
	f.mu.Lock()
	secret := f.secret
	f.mu.Unlock()
	if fd < 0 {
		return "", dbus.MakeFailedError(errors.New("no descriptor"))
	}
	w := os.NewFile(uintptr(fd), "secret")
	_, _ = w.WriteString(secret)
	_ = w.Close()
	return f.begin(sender, "RetrieveSecret", options)
}

func (s *settings) Read(string, string) (dbus.Variant, *dbus.Error) {
	(*fake)(s).mu.Lock()
	defer (*fake)(s).mu.Unlock()
	if (*fake)(s).notFound {
		return dbus.Variant{}, &dbus.Error{Name: notFoundIf}
	}
	return (*fake)(s).setting, nil
}

func (f *fake) set(change func(*fake)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	change(f)
}

func ctx5(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestResponseCodesMapToResults(t *testing.T) {
	f := startFake(t)
	for _, c := range []struct {
		code uint32
		want error
		fail bool
	}{{0, nil, false}, {1, ErrCancelled, true}, {2, nil, true}} {
		f.set(func(f *fake) {
			f.response, f.results = c.code, map[string]dbus.Variant{"autostart": dbus.MakeVariant(true)}
		})
		err := SetAutostart(ctx5(t), Autostart{Enable: true, Command: []string{"app", "--bg"}, Reason: "Start at login"})
		if (err != nil) != c.fail || (c.want != nil && !errors.Is(err, c.want)) {
			t.Errorf("code %d: %v", c.code, err)
		}
	}
	f.mu.Lock()
	opts := f.seen["RequestBackground"]
	f.mu.Unlock()
	if opts["reason"].Value() != "Start at login" || opts["dbus-activatable"].Value() != false {
		t.Errorf("options %v", opts)
	}
	f.set(func(f *fake) { f.response = 0 })
	if err := SetAutostart(ctx5(t), Autostart{Enable: true}); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	_, sentReason := f.seen["RequestBackground"]["reason"]
	f.mu.Unlock()
	if sentReason {
		t.Error("empty reason was sent")
	}
}

func TestAutostartNotGrantedIsNotAllowed(t *testing.T) {
	f := startFake(t)
	f.set(func(f *fake) { f.results = map[string]dbus.Variant{"autostart": dbus.MakeVariant(false)} })
	if err := SetAutostart(ctx5(t), Autostart{Enable: true}); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("enable: %v", err)
	}
	if err := SetAutostart(ctx5(t), Autostart{Enable: false}); err != nil {
		t.Fatalf("disable: %v", err)
	}
}

func TestCancelledContextClosesTheRequest(t *testing.T) {
	f := startFake(t)
	f.set(func(f *fake) { f.silent = true })
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := SetAutostart(ctx, Autostart{Enable: true}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	select {
	case h := <-f.closed:
		if !strings.Contains(string(h), "/request/") {
			t.Fatalf("closed %q", h)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Request.Close was not called")
	}
}

func TestContextBoundsADirectCall(t *testing.T) {
	startFake(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if _, err := LauncherExists(ctx, "x.desktop"); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("did not return promptly")
	}
}

func TestInstallLauncherInstallsTheBuiltEntry(t *testing.T) {
	f := startFake(t)
	f.set(func(f *fake) {
		f.results = map[string]dbus.Variant{"token": dbus.MakeVariant("t"), "name": dbus.MakeVariant("Renamed")}
	})
	err := InstallLauncher(ctx5(t), Launcher{ID: "org.example.App.x.desktop", Name: "Proposed", IconPNG: []byte{1}, Exec: []string{"app", "--play=a"}, Categories: []string{"Game"}})
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if !strings.Contains(f.entry, "Name=Renamed\n") || !strings.Contains(f.entry, `Exec="app" --play=a`) || f.seen["PrepareInstall"]["launcher_type"].Value() != uint32(1) {
		t.Fatalf("entry %q options %v", f.entry, f.seen["PrepareInstall"])
	}
}

func TestInstallLauncherNeedsAnInstallToken(t *testing.T) {
	startFake(t)
	if err := InstallLauncher(ctx5(t), Launcher{ID: "x.desktop", Name: "N", Exec: []string{"a"}}); err == nil {
		t.Fatal("want error for an empty token")
	}
}

func TestLauncherExistsTreatsGLibNoentAsMissing(t *testing.T) {
	f := startFake(t)
	if ok, err := LauncherExists(ctx5(t), "x.desktop"); err != nil || !ok {
		t.Fatalf("present: %v %v", ok, err)
	}
	f.set(func(f *fake) { f.missing = true })
	if ok, err := LauncherExists(ctx5(t), "x.desktop"); err != nil || ok {
		t.Fatalf("missing: %v %v", ok, err)
	}
	if err := UninstallLauncher(ctx5(t), "x.desktop"); err != nil {
		t.Fatalf("uninstalling a missing launcher: %v", err)
	}
}

func TestRetrieveSecretReadsThePipe(t *testing.T) {
	f := startFake(t)
	f.set(func(f *fake) { f.secret = "master-secret" })
	got, err := RetrieveSecret(ctx5(t))
	if err != nil || string(got) != "master-secret" {
		t.Fatalf("%q %v", got, err)
	}
	f.set(func(f *fake) { f.secret = "" })
	if _, err := RetrieveSecret(ctx5(t)); err == nil {
		t.Fatal("want error for an empty secret")
	}
}

func TestReadSetting(t *testing.T) {
	f := startFake(t)
	f.set(func(f *fake) { f.setting = dbus.MakeVariant(dbus.MakeVariant(uint32(2))) })
	if got, err := ReadSetting(ctx5(t), "org.freedesktop.appearance", "color-scheme"); err != nil || got != uint32(2) {
		t.Fatalf("%v %v", got, err)
	}
	f.set(func(f *fake) { f.notFound = true })
	if _, err := ReadSetting(ctx5(t), "org.freedesktop.appearance", "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

// TestLivePortal calls the real Secret portal, which answers only a sandboxed caller with an app id: run it by hand
// inside a Flatpak with DESKTOPENV_LIVE_PORTAL=1.
func TestLivePortal(t *testing.T) {
	if os.Getenv("DESKTOPENV_LIVE_PORTAL") != "1" {
		t.Skip("set DESKTOPENV_LIVE_PORTAL=1 inside a Flatpak to call the real Secret portal")
	}
	a, err := RetrieveSecret(ctx5(t))
	if err != nil || len(a) == 0 {
		t.Fatalf("first call: %q, %v", a, err)
	}
	b, err := RetrieveSecret(ctx5(t))
	if err != nil || string(a) != string(b) {
		t.Fatalf("second call must return the same secret: %v", err)
	}
}
