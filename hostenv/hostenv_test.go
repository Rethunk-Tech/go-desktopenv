package hostenv

import (
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestScrubAppImage(t *testing.T) {
	in := []string{
		"APPDIR=/tmp/.mount_x", "APPIMAGE=/home/u/App.AppImage", "OWD=/home/u",
		"LD_LIBRARY_PATH=/tmp/.mount_x/usr/lib:/tmp/.mount_x/usr/lib64:/opt/lib",
		"PATH=/tmp/.mount_x/usr/bin:/usr/bin",
		"GTK_PATH=/tmp/.mount_x//usr/lib/gtk-3.0",
		"HOME=/home/u",
	}
	want := []string{"LD_LIBRARY_PATH=/opt/lib", "PATH=/usr/bin", "HOME=/home/u"}
	if got := scrub(in, "/home/u/App.AppImage", "/tmp/.mount_x"); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := scrub(in, "", "/tmp/.mount_x"); !slices.Equal(got, in) {
		t.Fatalf("outside an AppImage the environment is untouched, got %v", got)
	}
}

func TestEnvironScrubsFromProcessEnv(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("only a Linux AppImage injects these variables, so Environ scrubs them only there")
	}
	t.Setenv("APPIMAGE", "/home/u/App.AppImage")
	t.Setenv("APPDIR", "/tmp/.mount_env")
	t.Setenv("HOSTENV_TEST_PATH", "/tmp/.mount_env/bin:/usr/bin")
	got := Environ()
	if !slices.Contains(got, "HOSTENV_TEST_PATH=/usr/bin") {
		t.Errorf("path entry under APPDIR kept: %v", got)
	}
	for _, kv := range got {
		if strings.HasPrefix(kv, "APPDIR=") || strings.HasPrefix(kv, "APPIMAGE=") {
			t.Errorf("%s survived", kv)
		}
	}
}
