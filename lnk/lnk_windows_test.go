//go:build windows

package lnk

import (
	"path/filepath"
	"testing"
)

func TestWriteThenArgumentsAndTarget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.lnk")
	s := Shortcut{Target: `C:\Windows\System32\cmd.exe`, Arguments: `/c "echo hi"`, Icon: `C:\Windows\System32\cmd.exe,0`, Description: "d"}
	if err := Write(path, s); err != nil {
		t.Fatal(err)
	}
	if got, err := Arguments(path); err != nil || got != s.Arguments {
		t.Errorf("got %q, %v", got, err)
	}
	if got, err := Target(path); err != nil || got != s.Target {
		t.Errorf("target %q, %v", got, err)
	}
	files, err := Files(filepath.Dir(path))
	if err != nil || len(files) != 1 {
		t.Errorf("files %v, %v", files, err)
	}
}
