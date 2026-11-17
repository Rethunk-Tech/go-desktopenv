package userdirs

import (
	"path/filepath"
	"testing"
)

func FuzzFlatpakApp(f *testing.F) {
	f.Add("h/u", "com.valvesoftware.Steam")
	f.Add("", "x")
	f.Add("h/u", "")
	f.Add("h/u", ".")
	f.Add("h/u", "..")
	f.Add("h/u", "../../.local/bin")
	f.Add("h/u", "a/b")
	f.Add("h/u", "/etc")
	f.Fuzz(func(t *testing.T, home, appID string) {
		got := FlatpakApp(home, appID)
		if got == "" {
			return
		}
		if home == "" {
			t.Fatalf("empty home produced %q", got)
		}
		root := filepath.Join(home, ".var", "app")
		rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(got))
		if err != nil || !filepath.IsLocal(rel) {
			t.Fatalf("FlatpakApp(%q, %q) = %q escapes %q", home, appID, got, root)
		}
		if rel != appID && rel != filepath.Clean(appID) {
			t.Fatalf("FlatpakApp(%q, %q) = %q, rel %q", home, appID, got, rel)
		}
	})
}
