package desktopentry

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestValue(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"plain":               "plain",
		`a\b`:                 `a\\b`,
		"a\tb":                `a\tb`,
		" lead":               `\slead`,
		"in side":             "in side",
		"a\nb\rc":             "a b c",
		"\nstart":             `\sstart`,
		`Mods\new` + "\nline": `Mods\\new line`,
	} {
		if got := Value(in); got != want {
			t.Errorf("Value(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExec(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		argv []string
		want string
	}{
		{[]string{"/usr/bin/x"}, `"/usr/bin/x"`},
		{[]string{"/x", "--play=a/b"}, `"/x" --play=a/b`},
		{[]string{"/x", ""}, `"/x" ""`},
		{[]string{"/x", "a b"}, `"/x" "a b"`},
		{[]string{"/home/u/Apps/50%off/M"}, `"/home/u/Apps/50%%off/M"`},
		{[]string{"/home/u/My Apps/$HOME/M"}, `"/home/u/My Apps/\\$HOME/M"`},
		{[]string{"/x", `q"z`}, `"/x" "q\\"z"`},
		{[]string{"/a\nb\tc/M"}, `"/a\nb\tc/M"`},
		{[]string{"/a\rb"}, `"/a\rb"`},
		{[]string{`/a\b`}, `"/a\\\\b"`},
	} {
		if got := Exec(c.argv...); got != c.want {
			t.Errorf("Exec(%q) = %s, want %s", c.argv, got, c.want)
		}
	}
}

// reserved is every character Exec quotes an argument for, plus the percent it doubles.
var reserved = []string{" ", "\t", "\n", "\r", `"`, "'", `\`, ">", "<", "~", "|", "&", ";", "$", "*", "?", "#", "(", ")", "`", "%", "%%"}

func TestSplitExecReadsWhatExecWrites(t *testing.T) {
	t.Parallel()
	cases := [][]string{
		{"/usr/bin/x"},
		{"/x", "plain", "", "--k=v"},
		{"/home/u/Apps/50%off/Mortar.AppImage", "/home/u/My Apps/$HOME/M", "/home/u/`x`\\y\"z/M", "/a\nb\tc/M"},
	}
	for _, r := range reserved {
		cases = append(cases, []string{"/p" + r + "q", "a" + r, r + "b"})
	}
	for _, argv := range cases {
		if got := SplitExec(Exec(argv...)); !reflect.DeepEqual(got, argv) {
			t.Errorf("SplitExec(%s) = %q, want %q", Exec(argv...), got, argv)
		}
	}
}

func TestSplitExecLeavesFieldCodesAndReadsOlderSpellings(t *testing.T) {
	t.Parallel()
	for in, want := range map[string][]string{
		`"/x" %u`:                   {"/x", "%u"},
		`"/x" %f %F`:                {"/x", "%f", "%F"},
		`"/opt/50%off/x" %u`:        {"/opt/50%off/x", "%u"},
		`/home/u/.local/bin/mortar`: {"/home/u/.local/bin/mortar"},
		`"/x" "" y`:                 {"/x", "", "y"},
		`   `:                       nil,
		`"/a\\nb"`:                  {`/a\nb`},
		`"/a\nb"`:                   {"/a\nb"},
		`"unclosed`:                 {"unclosed"},
	} {
		if got := SplitExec(in); !reflect.DeepEqual(got, want) {
			t.Errorf("SplitExec(%q) = %q, want %q", in, got, want)
		}
	}
}

// desktop-file-validate is the freedesktop reference check of the file format. CI installs it; elsewhere the test
// runs only where it is present.
func TestWrittenEntriesPassDesktopFileValidate(t *testing.T) {
	_, err := exec.LookPath("desktop-file-validate")
	if err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("desktop-file-validate is required in CI: install desktop-file-utils")
		}
		t.Skip("desktop-file-validate not on PATH")
	}
	for _, argv := range [][]string{
		{"/home/u/Apps/50%off/Mortar.AppImage", "--play=g/p"},
		{"/home/u/My Apps/$HOME/Mortar.AppImage"},
		{"/home/u/`x`\\y\"z/Mortar.AppImage", ""},
		{"/home/u/a\nb\tc/Mortar.AppImage"},
	} {
		body := "[Desktop Entry]\nType=Application\nName=" + Value(" a\\b\tc\nd") + "\nExec=" + Exec(argv...) + " %u\nTerminal=false\n"
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "x.desktop"), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		validate := exec.CommandContext(t.Context(), "desktop-file-validate", "x.desktop")
		validate.Dir = dir
		if out, err := validate.CombinedOutput(); err != nil {
			t.Errorf("%q: %v\n%s", argv, err, out)
		}
	}
}
