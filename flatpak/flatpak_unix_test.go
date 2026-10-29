//go:build unix

package flatpak

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// fakeSpawn puts a flatpak-spawn shell script first on PATH. It is a real process that stands in for the host
// bridge: it prints the lines of stdout on success.
func fakeSpawn(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "flatpak-spawn"), []byte("#!/bin/sh\n"+script), 0o700); err != nil { //nolint:gosec // G306: the test script must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// lookPathScript fails unless it gets exactly the argv HostLookPath promises, then prints answer and exits status.
func lookPathScript(answer string, status int) string {
	return `[ "$1" = --host ] && [ "$2" = sh ] && [ "$3" = -c ] && [ "$4" = 'command -v -- "$1"' ] && [ "$5" = sh ] && [ "$6" = steam ] || { echo "bad argv: $*" >&2; exit 99; }
echo '` + answer + `'
exit ` + strconv.Itoa(status) + "\n"
}

func TestHostLookPath(t *testing.T) {
	for name, c := range map[string]struct {
		answer string
		status int
		want   string
		found  bool
	}{
		"found":   {"/usr/bin/steam", 0, "/usr/bin/steam", true},
		"missing": {"", 1, "", false},
		"builtin": {"steam", 0, "", false},
	} {
		t.Run(name, func(t *testing.T) {
			fakeSpawn(t, lookPathScript(c.answer, c.status))
			got, err := HostLookPath(context.Background(), "steam")
			if c.found != (err == nil) || got != c.want || (!c.found && !errors.Is(err, exec.ErrNotFound)) {
				t.Fatalf("HostLookPath = %q, %v", got, err)
			}
		})
	}
}

func TestCommandSpawnsOnTheHostOnlyInsideAFlatpak(t *testing.T) {
	fakeSpawn(t, `echo "$@"`+"\n")
	t.Setenv("FLATPAK_ID", "org.example.App")
	cmd := Command(context.Background(), "steam", "-silent", "a b")
	if want := []string{"flatpak-spawn", "--host", "steam", "-silent", "a b"}; !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("inside: %q", cmd.Args)
	}
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "--host steam -silent a b" {
		t.Fatalf("inside output %q, %v", out, err)
	}

	t.Setenv("FLATPAK_ID", "")
	cmd = Command(context.Background(), "steam", "-silent")
	if want := []string{"steam", "-silent"}; !reflect.DeepEqual(cmd.Args, want) {
		t.Fatalf("outside: %q", cmd.Args)
	}
}
