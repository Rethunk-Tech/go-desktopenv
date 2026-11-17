package schemehandler

import (
	"strings"
	"testing"
)

func FuzzCommandLineToArgv(f *testing.F) {
	for _, s := range []string{
		`a "b c" d`,
		`"a\"b" c`,
		`a\\\"b`,
		`"C:\dir\\" x`,
		`a\b  ""`,
		"  tab\tseparated",
		`"C:\Vortex\Vortex.exe" "%1"`,
		`"C:\Program Files\Nexus Mods\Vortex.exe" "%1"`,
		"",
		`"`,
		`\`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, cmd string) {
		args := commandLineToArgv(cmd)
		if len(args) > len(cmd)+1 {
			t.Fatalf("commandLineToArgv(%q) returned %d args", cmd, len(args))
		}
		n := 0
		for _, a := range args {
			n += len(a)
		}
		if n > len(cmd) {
			t.Fatalf("commandLineToArgv(%q) args total %d bytes from %d-byte input", cmd, n, len(cmd))
		}
	})
}

func FuzzCheckLink(f *testing.F) {
	for _, s := range []string{
		"nxm://skyrim/mods/1/files/2",
		"http://example.com",
		"https://x",
		"--gpu-launcher=sh",
		"-x",
		"",
		"/etc/passwd",
		"1http://x",
		"nxm:",
		"://x",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, link string) {
		err := checkLink(link)
		if err != nil {
			return
		}
		scheme, _, ok := strings.Cut(link, ":")
		if !ok || checkScheme(scheme) != nil {
			t.Fatalf("checkLink accepted %q", link)
		}
	})
}

func FuzzSplitPrevious(f *testing.F) {
	f.Add(`"C:\Vortex\Vortex.exe" "%1"`, `C:\Vortex\Vortex.exe,0`, "URL:Vortex")
	f.Add("cmd", "", "")
	f.Add("", "icon", "name")
	f.Add("a\nb", "c", "d")
	f.Fuzz(func(t *testing.T, cmd, icon, name string) {
		if strings.Contains(cmd, "\n") || strings.Contains(icon, "\n") {
			return
		}
		saved := previousID(cmd, icon, name)
		got := splitPrevious(saved)
		if cmd == "" {
			if saved != "" || got != (previous{}) {
				t.Fatalf("empty command: saved %q got %+v", saved, got)
			}
			return
		}
		if got.cmd != cmd || got.icon != icon || got.name != name || !got.hasIcon || !got.hasName {
			t.Fatalf("splitPrevious(previousID(%q, %q, %q)) = %+v", cmd, icon, name, got)
		}
	})
}

func FuzzEscapeForCmd(f *testing.F) {
	for _, s := range []string{
		"nxm://stardewvalley/mods/21297/files/90031?key=abc-DEF_1&expires=1700000000&user_id=123",
		"nxm://g/mods/1&calc|whoami>x",
		`nxm://g/1"&calc`,
		"nxm://g/1\r\ncalc",
		"nxm://g/%PATH%",
		"nxm://g/1 2",
		"nxm://g/mods/1?a=%20b",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, link string) {
		out, err := escapeForCmd(link)
		if err != nil {
			return
		}
		if strings.ContainsAny(out, "\"\r\n \t") {
			t.Fatalf("escapeForCmd(%q) = %q holds a character cmd.exe cannot take as one word", link, out)
		}
		for i := 0; i < len(out); i++ {
			c := out[i]
			if c < 0x20 {
				t.Fatalf("escapeForCmd(%q) = %q holds a control byte", link, out)
			}
			if c == '%' && i+1 < len(out) && (out[i+1] == '_' || out[i+1] >= 'A' && out[i+1] <= 'Z' || out[i+1] >= 'a' && out[i+1] <= 'z') {
				t.Fatalf("escapeForCmd(%q) = %q holds a %% cmd.exe would expand", link, out)
			}
			if c == '^' {
				if i+1 >= len(out) {
					t.Fatalf("escapeForCmd(%q) = %q ends on a caret", link, out)
				}
				i++
				continue
			}
			if strings.IndexByte("&|<>()!", c) >= 0 {
				t.Fatalf("escapeForCmd(%q) = %q left %q unescaped", link, out, c)
			}
		}
	})
}

func FuzzCheckScheme(f *testing.F) {
	for _, s := range []string{"nxm", "http", "https", "h323", "a+b.c-1", "", "1http", "x_y", "NXM"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, scheme string) {
		err := checkScheme(scheme)
		if err != nil {
			return
		}
		if scheme == "" {
			t.Fatal("checkScheme accepted an empty scheme")
		}
		if c := scheme[0]; c < 'A' || c > 'Z' && c < 'a' || c > 'z' {
			t.Fatalf("checkScheme accepted %q with a non-letter start", scheme)
		}
	})
}
