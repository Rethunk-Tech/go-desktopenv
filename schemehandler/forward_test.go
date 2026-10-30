package schemehandler

import (
	"reflect"
	"testing"
)

func TestWindowsForwardArgv(t *testing.T) {
	t.Parallel()
	link := "nxm://skyrim/mods/1/files/2"
	for name, c := range map[string]struct {
		cmd      string
		wantName string
		wantArgs []string
	}{
		"quoted field":          {`"C:\Vortex\Vortex.exe" "%1"`, `C:\Vortex\Vortex.exe`, []string{link}},
		"spaces in the program": {`"C:\Program Files\Nexus Mods\Vortex.exe" "%1"`, `C:\Program Files\Nexus Mods\Vortex.exe`, []string{link}},
		"no field code":         {`"C:\Vortex\Vortex.exe" --nxm`, `C:\Vortex\Vortex.exe`, []string{"--nxm", link}},
	} {
		got, args, err := windowsForwardArgv(previousID(c.cmd, `C:\Vortex\Vortex.exe,0`, "URL:Vortex"), link)
		if err != nil || got != c.wantName || !reflect.DeepEqual(args, c.wantArgs) {
			t.Errorf("%s: argv = %q %v, %v", name, got, args, err)
		}
	}
	if _, _, err := windowsForwardArgv("", link); err == nil {
		t.Error("an empty command forwarded")
	}
}

func TestLinuxForwardArgv(t *testing.T) {
	t.Parallel()
	link := "nxm://skyrim/mods/1"
	for exec, want := range map[string][]string{
		`"/usr/bin/flatpak" run com.nexusmods.vortex %u`: {"/usr/bin/flatpak", "run", "com.nexusmods.vortex", link},
		`"/usr/lib/nexus mods/vortex" --url %u`:          {"/usr/lib/nexus mods/vortex", "--url", link},
		// The spec reads quoted and unquoted runs with no space between them as one argument.
		`"/home/u/My Apps"/App.AppImage --nxm %u`: {"/home/u/My Apps/App.AppImage", "--nxm", link},
		`/usr/bin/vortex --nxm`:                   {"/usr/bin/vortex", "--nxm", link},
		`/usr/bin/vortex %f %U`:                   {"/usr/bin/vortex", link},
	} {
		name, args, err := linuxForwardArgv("vortex.desktop", link, func(id string) (string, error) {
			if id != "vortex.desktop" {
				t.Errorf("id = %q", id)
			}
			return exec, nil
		})
		if err != nil || !reflect.DeepEqual(append([]string{name}, args...), want) {
			t.Errorf("%s: argv = %q %v, %v", exec, name, args, err)
		}
	}
	if _, _, err := linuxForwardArgv("x.desktop", link, func(string) (string, error) { return "  ", nil }); err == nil {
		t.Error("an empty Exec forwarded")
	}
	if _, _, err := linuxForwardArgv("", link, nil); err == nil {
		t.Error("no previous handler forwarded")
	}
}

func TestForwardArgvKeepsMetacharactersInOneElement(t *testing.T) {
	t.Parallel()
	link := `nxm://othergame/mods/1?x=$(touch /tmp/pwned);id&` + "`uname`" + ` "quoted" spaced`
	name, args, err := windowsForwardArgv(previousID(`"C:\Vortex\Vortex.exe" "%1"`, "", ""), link)
	if err != nil || name != `C:\Vortex\Vortex.exe` || len(args) != 1 || args[0] != link {
		t.Fatalf("windows argv = %q %v, %v", name, args, err)
	}
	name, args, err = linuxForwardArgv("vortex.desktop", link, func(string) (string, error) { return `/usr/bin/vortex %u`, nil })
	if err != nil || name != "/usr/bin/vortex" || len(args) != 1 || args[0] != link {
		t.Fatalf("linux argv = %q %v, %v", name, args, err)
	}
}

func TestCommandLineToArgv(t *testing.T) {
	t.Parallel()
	for in, want := range map[string][]string{
		`a "b c" d`:        {"a", "b c", "d"},
		`"a\"b" c`:         {`a"b`, "c"},
		`a\\\"b`:           {`a\"b`},
		`"C:\dir\\" x`:     {`C:\dir\`, "x"},
		`a\b  ""`:          {`a\b`, ""},
		"  tab\tseparated": {"tab", "separated"},
	} {
		if got := commandLineToArgv(in); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %q, want %q", in, got, want)
		}
	}
}

func TestPreviousRoundTrip(t *testing.T) {
	t.Parallel()
	cmd, icon, name := `"C:\Vortex\Vortex.exe" "%1"`, `C:\Vortex\Vortex.exe,0`, "URL:Vortex"
	want := previous{cmd: cmd, icon: icon, name: name, hasIcon: true, hasName: true}
	if got := splitPrevious(previousID(cmd, icon, name)); got != want {
		t.Fatalf("splitPrevious = %+v, want %+v", got, want)
	}
	if got := splitPrevious(cmd + "\n" + icon); got != (previous{cmd: cmd, icon: icon, hasIcon: true}) {
		t.Fatalf("two-line value = %+v", got)
	}
	if previousID("", icon, name) != "" || splitPrevious("") != (previous{}) {
		t.Fatal("no command means nothing saved")
	}
}
