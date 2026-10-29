package portal

import (
	"regexp"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestHandlePathFromAUniqueName(t *testing.T) {
	t.Parallel()
	if got, want := handlePath(":1.42", "tok"), dbus.ObjectPath("/org/freedesktop/portal/desktop/request/1_42/tok"); got != want {
		t.Fatalf("%q, want %q", got, want)
	}
}

// A portal handle token is a D-Bus object path element, fresh on each call.
func TestNewTokenIsFreshAndAPathElement(t *testing.T) {
	t.Parallel()
	element := regexp.MustCompile(`^go_desktopenv_launcher_[a-z0-9]+$`)
	a, b := newToken("launcher"), newToken("launcher")
	if a == b || !element.MatchString(a) || !element.MatchString(b) {
		t.Fatalf("tokens %q, %q", a, b)
	}
}

func TestDesktopEntryEscapesWhatTheFormatReads(t *testing.T) {
	t.Parallel()
	got := desktopEntry(Launcher{
		Name:       " Mods\\new\nline\rend\tx",
		Comment:    "a\\b\nc",
		Exec:       []string{"/opt/my app/x", "--k=a\\b\nc\td"},
		Categories: []string{"Game", "A;B", `C\D`},
	})
	for _, want := range []string{
		"[Desktop Entry]\nType=Application\n",
		`Name=\sMods\\new line end\tx` + "\n",
		`Comment=a\\b c` + "\n",
		`Exec="/opt/my app/x" "--k=a\\\\b\nc\td"` + "\n",
		"Terminal=false\n",
		`Categories=Game;A\;B;C\\D;` + "\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if bare := desktopEntry(Launcher{Name: "N", Exec: []string{"x"}}); strings.Contains(bare, "Comment=") || strings.Contains(bare, "Categories=") {
		t.Errorf("optional keys written empty:\n%s", bare)
	}
}

func TestUnwrapRemovesEveryVariant(t *testing.T) {
	t.Parallel()
	for name, in := range map[string]any{
		"plain": uint32(3),
		"v":     dbus.MakeVariant(uint32(3)),
		"vv":    dbus.MakeVariant(dbus.MakeVariant(uint32(3))),
	} {
		if got := unwrap(in); got != uint32(3) {
			t.Errorf("%s: %v", name, got)
		}
	}
}
