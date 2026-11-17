package fonts

import (
	"strings"
	"testing"
)

func FuzzParseFcList(f *testing.F) {
	f.Add("DejaVu Sans,DejaVu Sans Condensed\nOpen Sans\ndejavu sans\n\nNoto Serif,Noto Serif CJK\n")
	f.Add("")
	f.Add("  \n,")
	f.Add("Arial\nArial\nARIAL")
	f.Fuzz(func(t *testing.T, out string) {
		got := ParseFcList(out)
		n, seen := 0, map[string]struct{}{}
		for line := range strings.SplitSeq(out, "\n") {
			first, _, _ := strings.Cut(line, ",")
			first = strings.TrimSpace(first)
			if first == "" {
				continue
			}
			n++
			seen[strings.ToLower(first)] = struct{}{}
		}
		assertTidy(t, got, n, seen)
	})
}

func FuzzParseWindowsNames(f *testing.F) {
	f.Add("Arial (TrueType)\nArial Bold Italic (TrueType)\nSegoe UI Light & Segoe UI Semilight (TrueType)\nSegoe UI (TrueType)\nBlack (OpenType)")
	f.Add("")
	f.Add("Bold (TrueType)")
	f.Add("Foo & Bar & Baz (TrueType)")
	f.Fuzz(func(t *testing.T, block string) {
		var names []string
		for line := range strings.SplitSeq(block, "\n") {
			names = append(names, line)
		}
		got := ParseWindowsNames(names)
		n := 0
		for _, name := range names {
			if before, _, found := strings.CutLast(name, " ("); found {
				name = before
			}
			for part := range strings.SplitSeq(name, " & ") {
				if strings.Join(strings.Fields(part), " ") != "" {
					n++
				}
			}
		}
		if len(got) > n {
			t.Fatalf("more families than input names: %d > %d", len(got), n)
		}
		for i, s := range got {
			if s == "" {
				t.Fatal("empty family")
			}
			if i == 0 {
				continue
			}
			if strings.ToLower(got[i-1]) > strings.ToLower(s) {
				t.Fatalf("unsorted: %q then %q", got[i-1], s)
			}
			if strings.EqualFold(got[i-1], s) {
				t.Fatalf("duplicate %q", s)
			}
		}
	})
}

func assertTidy(t *testing.T, got []string, n int, seen map[string]struct{}) {
	t.Helper()
	if len(got) > n {
		t.Fatalf("more families than input names: %d > %d", len(got), n)
	}
	for i, s := range got {
		if s == "" {
			t.Fatal("empty family")
		}
		if _, ok := seen[strings.ToLower(s)]; !ok {
			t.Fatalf("family %q was not in the input", s)
		}
		if i == 0 {
			continue
		}
		if strings.ToLower(got[i-1]) > strings.ToLower(s) {
			t.Fatalf("unsorted: %q then %q", got[i-1], s)
		}
		if strings.EqualFold(got[i-1], s) {
			t.Fatalf("duplicate %q", s)
		}
	}
}
