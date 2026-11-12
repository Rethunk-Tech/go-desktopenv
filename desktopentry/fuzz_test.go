package desktopentry

import (
	"strings"
	"testing"
)

func FuzzSplitExec(f *testing.F) {
	for _, s := range []string{
		`"/usr/bin/app"`,
		`"/usr/bin/app" %u`,
		`"/usr/bin/app" %f %F %u %U %d %D %n %N %i %c %k %v %m`,
		`"/usr/bin/app" "%%u"`,
		Exec("/usr/bin/app"),
		Exec("/usr/bin/app", "hello world"),
		Exec("/path/with space/app", "", "a%b"),
		Exec("/usr/bin/app", "tab\there", "line\nbreak", "ret\r"),
		Exec(`C:\Program Files\app`, `"quoted"`, `$HOME`, "`tick`", "a|b", "c&d"),
		"",
		`"`,
		`\`,
		`""`,
		`"app" foo`,
		"a\\\nb",
		`"/bin/sh" -c "echo hi"`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, value string) {
		args := SplitExec(value)
		if len(args) == 0 {
			return
		}
		got := SplitExec(Exec(args...))
		if len(got) != len(args) {
			t.Fatalf("SplitExec(Exec(SplitExec(%q))) length: got %d want %d\n%q\n%q", value, len(got), len(args), got, args)
		}
		for i := range args {
			if got[i] != args[i] {
				t.Fatalf("SplitExec(Exec(SplitExec(%q)))[%d]: got %q want %q", value, i, got[i], args[i])
			}
		}
	})
}

func FuzzExec(f *testing.F) {
	f.Add("/usr/bin/app", "hello world", "%u")
	f.Add("", "", "")
	f.Add("/path/with space/app", "a%b", `\$"`)
	f.Add("app", "\t", "\n")
	f.Add(`C:\Program Files\app`, "a|b", "c;d")
	f.Fuzz(func(t *testing.T, a, b, c string) {
		in := []string{a, b, c}
		got := SplitExec(Exec(in...))
		if len(got) != 3 {
			t.Fatalf("SplitExec(Exec(%q)) length: got %d\n%q", in, len(got), got)
		}
		for i := range in {
			if got[i] != in[i] {
				t.Fatalf("SplitExec(Exec(%q))[%d]: got %q want %q", in, i, got[i], in[i])
			}
		}
	})
}

func FuzzSetKey(f *testing.F) {
	f.Add("[Desktop Entry]\nName=Foo\nExec=bar\n", "Name", "Bar")
	f.Add("[Desktop Entry]\nName=Foo\n", "Exec", Exec("/usr/bin/app", "%u"))
	f.Add("Name=Foo", "Name", "Bar")
	f.Add("Name=Foo\n", "Name", Value(" leading"))
	f.Add("", "Name", "x")
	f.Add("[Desktop Entry]\n", "Name", "x")
	f.Fuzz(func(t *testing.T, entry, key, value string) {
		if strings.ContainsAny(key, "=\n") || strings.Contains(value, "\n") {
			return
		}
		out, err := SetKey(entry, key, value)
		if err != nil {
			return
		}
		prefix := key + "="
		found := false
		for _, line := range strings.SplitAfter(out, "\n") {
			if strings.HasPrefix(line, prefix) {
				got := strings.TrimSuffix(line, "\n")
				if got != prefix+value {
					t.Fatalf("SetKey first %q line: got %q want %q", key, got, prefix+value)
				}
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("SetKey(%q, %q, %q) succeeded but wrote no %s= line", entry, key, value, key)
		}
	})
}
