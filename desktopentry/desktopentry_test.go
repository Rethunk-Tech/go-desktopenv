package desktopentry

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
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

// desktop-file-validate is the freedesktop reference check of the file format. CI's Linux legs install it; elsewhere
// the test runs only where it is present.
func TestWrittenEntriesPassDesktopFileValidate(t *testing.T) {
	_, err := exec.LookPath("desktop-file-validate")
	if err != nil {
		if os.Getenv("CI") != "" && runtime.GOOS == "linux" {
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
		body := Entry{
			{"Type", "Application"},
			{"Name", Value(" a\\b\tc\nd")},
			{"Exec", Exec(argv...) + " %u"},
			{"Terminal", "false"},
		}.Marshal()
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

func TestMarshalWritesKeysInOrder(t *testing.T) {
	t.Parallel()
	exe, arg, name := "/usr/bin/app", "--play=g/p", "Profile (Game)"
	got := Entry{
		{"Type", "Application"},
		{"Name", Value(name)},
		{"Comment", "Play this profile"},
		{"Exec", Exec(exe, arg)},
		{"Icon", "app"},
		{"Terminal", "false"},
		{"Categories", "Game;"},
	}.Marshal()
	want := "[Desktop Entry]\nType=Application\nName=" + Value(name) + "\nComment=Play this profile\nExec=" + Exec(exe, arg) + "\nIcon=app\nTerminal=false\nCategories=Game;\n"
	if got != want {
		t.Errorf("Marshal() = %q, want %q", got, want)
	}
}

func TestMarshalWritesAutostartKeys(t *testing.T) {
	t.Parallel()
	exe := "/usr/bin/app"
	got := Entry{
		{"Type", "Application"},
		{"Name", "App"},
		{"Exec", Exec(exe)},
		{"X-GNOME-Autostart-enabled", "true"},
	}.Marshal()
	want := "[Desktop Entry]\nType=Application\nName=App\nExec=" + Exec(exe) + "\nX-GNOME-Autostart-enabled=true\n"
	if got != want {
		t.Errorf("Marshal() = %q, want %q", got, want)
	}
}

func TestSetKeyRewritesNameAndKeepsUnknownKeys(t *testing.T) {
	t.Parallel()
	in := Entry{
		{"Type", "Application"},
		{"Name", Value("old")},
		{"Comment", "Play this profile"},
		{"Exec", Exec("/usr/bin/app", "--play=g/p")},
		{"X-Custom", "keep"},
	}.Marshal()
	got, err := SetKey(in, "Name", Value("new (game)"))
	if err != nil {
		t.Fatal(err)
	}
	want := Entry{
		{"Type", "Application"},
		{"Name", Value("new (game)")},
		{"Comment", "Play this profile"},
		{"Exec", Exec("/usr/bin/app", "--play=g/p")},
		{"X-Custom", "keep"},
	}.Marshal()
	if got != want {
		t.Errorf("SetKey Name = %q, want %q", got, want)
	}
}

func TestSetKeyRewritesExecAndKeepsUnknownKeys(t *testing.T) {
	t.Parallel()
	in := Entry{
		{"Type", "Application"},
		{"Name", "Profile"},
		{"Exec", Exec("/old/app", "--play=g/p")},
		{"X-Custom", "keep"},
	}.Marshal()
	got, err := SetKey(in, "Exec", Exec("/new/app", "--play=g/p"))
	if err != nil {
		t.Fatal(err)
	}
	want := Entry{
		{"Type", "Application"},
		{"Name", "Profile"},
		{"Exec", Exec("/new/app", "--play=g/p")},
		{"X-Custom", "keep"},
	}.Marshal()
	if got != want {
		t.Errorf("SetKey Exec = %q, want %q", got, want)
	}
}

func TestSetKeyKeepsAMissingTrailingNewline(t *testing.T) {
	t.Parallel()
	got, err := SetKey("[Desktop Entry]\nName=old", "Name", "new")
	if err != nil {
		t.Fatal(err)
	}
	if got != "[Desktop Entry]\nName=new" {
		t.Errorf("SetKey = %q, want %q", got, "[Desktop Entry]\nName=new")
	}
}

func TestSetKeyRewritesOnlyTheFirstMatch(t *testing.T) {
	t.Parallel()
	got, err := SetKey("Name=a\nName=b\n", "Name", "c")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Name=c\nName=b\n" {
		t.Errorf("SetKey = %q, want %q", got, "Name=c\nName=b\n")
	}
}

func TestSetKeyMissingKey(t *testing.T) {
	t.Parallel()
	_, err := SetKey("[Desktop Entry]\nType=Application\n", "Name", "x")
	if err == nil {
		t.Fatal("SetKey missing key: got nil, want error")
	}
}
