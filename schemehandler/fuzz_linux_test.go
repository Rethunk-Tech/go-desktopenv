package schemehandler

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-desktopenv/desktopentry"
)

func FuzzExecTarget(f *testing.F) {
	f.Add([]byte("[Desktop Entry]\nExec=\"/usr/bin/app\" %u\n"))
	f.Add([]byte("Exec=\"/usr/bin/flatpak\" run com.nexusmods.vortex %u\n"))
	f.Add([]byte("Exec=/usr/bin/vortex --nxm\n"))
	f.Add([]byte("Exec=\"\n"))
	f.Add([]byte(""))
	f.Fuzz(func(t *testing.T, entry []byte) {
		got := execTarget(entry)
		if got == "" {
			return
		}
		if len(got) > len(entry) {
			t.Fatalf("execTarget longer than the file: %q", got)
		}
		var found bool
		for line := range strings.SplitSeq(string(entry), "\n") {
			rest, ok := strings.CutPrefix(line, "Exec=")
			if !ok || !strings.HasPrefix(rest, `"`) {
				continue
			}
			args := desktopentry.SplitExec(rest)
			if len(args) == 0 || args[0] != got {
				t.Fatalf("execTarget(%q) = %q, SplitExec = %q", entry, got, args)
			}
			found = true
			break
		}
		if !found {
			t.Fatalf("execTarget(%q) = %q from no Exec= line", entry, got)
		}
	})
}

func FuzzListsMime(f *testing.F) {
	f.Add([]byte("[Desktop Entry]\nMimeType=x-scheme-handler/nxm;application/x-app;\n"), "x-scheme-handler/nxm")
	f.Add([]byte("MimeType=x-scheme-handler/nxmextra;\n"), "x-scheme-handler/nxm")
	f.Add([]byte(""), "x-scheme-handler/nxm")
	f.Fuzz(func(t *testing.T, entry []byte, mime string) {
		got := listsMime(entry, mime)
		if !got {
			return
		}
		var listed bool
		for line := range strings.SplitSeq(string(entry), "\n") {
			v, ok := strings.CutPrefix(line, "MimeType=")
			if !ok {
				continue
			}
			for item := range strings.SplitSeq(strings.TrimSuffix(v, ";"), ";") {
				if strings.TrimSpace(item) == mime {
					listed = true
				}
			}
			break
		}
		if !listed {
			t.Fatalf("listsMime(%q, %q) is true, but mime is not an item", entry, mime)
		}
	})
}

func FuzzCheckDesktopID(f *testing.F) {
	for _, s := range []string{"org.example.App.desktop", "app.desktop", "", ".", "..", ".desktop", "a/b.desktop", `a\b.desktop`, "foo"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, id string) {
		err := checkDesktopID(id)
		if err != nil {
			return
		}
		if id != filepath.Base(id) || strings.ContainsAny(id, `/\`) || !strings.HasSuffix(id, ".desktop") || id == ".desktop" {
			t.Fatalf("checkDesktopID accepted %q", id)
		}
		dir := "applications"
		got := filepath.Join(dir, id)
		rel, err := filepath.Rel(dir, got)
		if err != nil || !filepath.IsLocal(rel) {
			t.Fatalf("DesktopID %q escapes the applications folder as %q", id, got)
		}
	})
}
