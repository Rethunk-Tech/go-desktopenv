package locale

import (
	"strings"
	"testing"
)

func FuzzFromPOSIX(f *testing.F) {
	for _, s := range []string{"de_DE.UTF-8@euro", "en_US.UTF-8", "fr", "C", "POSIX", "C.UTF-8", "", "zh_CN.UTF-8", "en@quot"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, v string) {
		got := FromPOSIX(v)
		if strings.ContainsAny(got, ".@") {
			t.Fatalf("FromPOSIX(%q) = %q still holds a modifier", v, got)
		}
		stripped := v
		if i := strings.IndexAny(v, ".@"); i >= 0 {
			stripped = v[:i]
		}
		if stripped == "C" || stripped == "POSIX" {
			if got != "" {
				t.Fatalf("FromPOSIX(%q) = %q, want empty for C/POSIX", v, got)
			}
			return
		}
		if want := strings.ReplaceAll(stripped, "_", "-"); got != want {
			t.Fatalf("FromPOSIX(%q) = %q, want %q", v, got, want)
		}
	})
}
