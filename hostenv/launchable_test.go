package hostenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLaunchablePrefersImageOverMount(t *testing.T) {
	dir := t.TempDir()
	img := filepath.Join(dir, "App.AppImage")
	if err := os.WriteFile(img, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	mount := filepath.Join(dir, "mnt")
	t.Setenv("APPIMAGE", img)
	t.Setenv("APPDIR", mount)
	if got := Launchable(filepath.Join(mount, "usr", "bin", "app")); got != img {
		t.Fatalf("got %q, want the image %q", got, img)
	}
	if got := Launchable("/usr/bin/app"); got != "/usr/bin/app" {
		t.Fatalf("outside the mount must be unchanged, got %q", got)
	}
}

func TestLaunchableIgnoresAForeignOrNonFileImage(t *testing.T) {
	dir := t.TempDir()
	mount := filepath.Join(dir, "mnt")
	exe := filepath.Join(mount, "usr", "bin", "app")
	t.Setenv("APPDIR", mount)

	t.Setenv("APPIMAGE", dir)
	if got := Launchable(exe); got != exe {
		t.Fatalf("a folder is not an image, got %q", got)
	}
	img := filepath.Join(dir, "Other.AppImage")
	if err := os.WriteFile(img, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APPIMAGE", img)
	other := filepath.Join(dir, "native", "app")
	if got := Launchable(other); got != other {
		t.Fatalf("an exe outside APPDIR must not be swapped for the inherited APPIMAGE, got %q", got)
	}
}
