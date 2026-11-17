package hostenv

import (
	"path"
	"strings"
	"testing"
)

func FuzzScrub(f *testing.F) {
	f.Add("APPDIR=/tmp/.mount_x\nAPPIMAGE=/home/u/App.AppImage\nOWD=/home/u\nLD_LIBRARY_PATH=/tmp/.mount_x/usr/lib:/opt/lib\nPATH=/tmp/.mount_x/usr/bin:/usr/bin\nHOME=/home/u", "/home/u/App.AppImage", "/tmp/.mount_x")
	f.Add("PATH=/usr/bin", "", "/tmp/.mount_x")
	f.Add("PATH=/usr/bin", "/home/u/App.AppImage", "")
	f.Add("NOEQUALS\nFOO=bar", "/a", "/b")
	f.Fuzz(func(t *testing.T, envBlock, appimage, appdir string) {
		var env []string
		for line := range strings.SplitSeq(envBlock, "\n") {
			env = append(env, line)
		}
		got := scrub(env, appimage, appdir)
		if len(got) > len(env) {
			t.Fatalf("scrub grew the environment from %d to %d", len(env), len(got))
		}
		if appimage == "" || appdir == "" {
			if strings.Join(got, "\n") != strings.Join(env, "\n") {
				t.Fatalf("outside an AppImage scrub changed the environment")
			}
			return
		}
		root := path.Clean(appdir)
		for _, kv := range got {
			k, v, ok := strings.Cut(kv, "=")
			if !ok {
				t.Fatalf("scrub kept %q, which has no =", kv)
			}
			switch k {
			case "APPDIR", "APPIMAGE", "OWD":
				t.Fatalf("%s survived", kv)
			}
			if !strings.Contains(v, root) {
				continue
			}
			for p := range strings.SplitSeq(v, ":") {
				clean := path.Clean(p)
				if p == root || strings.HasPrefix(clean, root+"/") {
					t.Fatalf("path under APPDIR kept: %q in %q", p, kv)
				}
			}
		}
	})
}
