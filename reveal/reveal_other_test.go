//go:build !windows

package reveal

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenStartsTheFileManager(t *testing.T) {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	bin, err := exec.LookPath("true")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(bin, filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if err := Open(t.TempDir()); err != nil {
		t.Fatal(err)
	}
}
