package lnk

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestFilesListsOnlyLnk(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.lnk", "B.LNK", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "d.lnk"), 0o750); err != nil {
		t.Fatal(err)
	}
	got, err := Files(dir)
	want := []string{filepath.Join(dir, "B.LNK"), filepath.Join(dir, "a.lnk")}
	slices.Sort(got)
	slices.Sort(want)
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("got %v, %v", got, err)
	}
	if got, err := Files(filepath.Join(dir, "missing")); got != nil || err != nil {
		t.Errorf("missing dir: %v, %v", got, err)
	}
}
