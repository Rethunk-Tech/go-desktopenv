package hostfile

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("HOSTFILE_HELPER") == "1" {
		_, _ = os.Stdout.WriteString("from-child")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestExecRunsAProcess(t *testing.T) {
	t.Setenv("HOSTFILE_HELPER", "1")
	out, err := Exec(os.Args[0])
	if err != nil || out != "from-child" {
		t.Fatalf("got %q, %v", out, err)
	}
}

func TestConfigHome(t *testing.T) {
	home, err := os.UserHomeDir() // userdirs honours XDG_CONFIG_HOME only for the real home
	if err != nil {
		t.Skip(err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))
	t.Setenv("FLATPAK_ID", "")
	if got := ConfigHome(home); got != os.Getenv("XDG_CONFIG_HOME") {
		t.Errorf("outside a Flatpak got %q", got)
	}
	t.Setenv("FLATPAK_ID", "org.example.App")
	if got := ConfigHome(home); got != filepath.Join(home, ".config") {
		t.Errorf("inside a Flatpak got %q", got)
	}
}

func TestHostOutsideFlatpakUsesFiles(t *testing.T) {
	t.Setenv("FLATPAK_ID", "")
	dir := t.TempDir()
	h := Host{Run: func(name string, args ...string) (string, error) {
		return name + " " + strings.Join(args, " "), nil
	}}
	if out, _ := h.Command("/usr/bin/tool", "a"); out != "/usr/bin/tool a" {
		t.Errorf("Command got %q", out)
	}
	if _, err := h.Read(filepath.Join(dir, "none")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Read of a missing file: %v", err)
	}
	f := filepath.Join(dir, "sub", "f")
	if err := h.Write(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if b, err := h.Read(f); err != nil || string(b) != "x" {
		t.Errorf("Read got %q, %v", b, err)
	}
	if !h.IsDir(dir) || h.IsDir(f) {
		t.Error("IsDir")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(f, link); err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.EvalSymlinks(f)
	if got := h.Resolve(link); got != want {
		t.Errorf("Resolve got %q", got)
	}
	if got := h.Resolve(filepath.Join(dir, "none")); got != filepath.Join(dir, "none") {
		t.Errorf("Resolve of a missing path got %q", got)
	}
	if err := h.Remove(f); err != nil {
		t.Fatal(err)
	}
	if err := h.Remove(f); err != nil {
		t.Errorf("Remove of a missing file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "keep"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.Remove(filepath.Join(dir, "sub")); err == nil {
		t.Error("Remove of a non-empty folder should fail")
	}
}

func TestHostInsideFlatpakRunsHostScripts(t *testing.T) {
	t.Setenv("FLATPAK_ID", "org.example.App")
	exit3 := exec.CommandContext(context.Background(), "sh", "-c", "exit 3").Run()
	var calls [][]string
	reply, fail := "out\n", error(nil)
	h := Host{Run: func(name string, args ...string) (string, error) {
		calls = append(calls, append([]string{name}, args...))
		return reply, fail
	}}

	if got := h.Resolve("/p"); got != "out" {
		t.Errorf("Resolve got %q", got)
	}
	reply = ""
	if got := h.Resolve("/p"); got != "/p" {
		t.Errorf("Resolve with no answer got %q", got)
	}
	fail = errors.New("boom")
	if got := h.Resolve("/p"); got != "/p" {
		t.Errorf("Resolve on error got %q", got)
	}
	if h.IsDir("/p") {
		t.Error("IsDir should be false when the script fails")
	}
	if err := h.Write("/p", []byte("b"), 0o600); err == nil {
		t.Error("Write should pass the error on")
	}
	if err := h.Remove("/p"); err == nil {
		t.Error("Remove should pass the error on")
	}
	if _, err := h.Read("/p"); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Read of a failing script: %v", err)
	}
	fail = exit3
	if _, err := h.Read("/p"); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("exit 3 should be ErrNotExist, got %v", err)
	}
	fail, reply = nil, "data"
	if b, err := h.Read("/p"); err != nil || string(b) != "data" {
		t.Errorf("Read got %q, %v", b, err)
	}
	if !h.IsDir("/p") || h.Write("/p", nil, 0o600) != nil || h.Remove("/p") != nil {
		t.Error("successful scripts")
	}
	for _, c := range calls {
		if c[0] != "flatpak-spawn" || !slices.Contains(c, "--host") {
			t.Fatalf("not spawned on the host: %v", c)
		}
	}
}
