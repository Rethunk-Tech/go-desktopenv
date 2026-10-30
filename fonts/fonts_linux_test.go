package fonts

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestListReadsFcList(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\nprintf 'Open Sans\\nDejaVu Sans,DejaVu Sans Mono\\n'\n"
	if err := os.WriteFile(filepath.Join(dir, "fc-list"), []byte(script), 0o700); err != nil { //nolint:gosec // G306: the test script must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got, err := List()
	if want := []string{"DejaVu Sans", "Open Sans"}; err != nil || !slices.Equal(got, want) {
		t.Fatalf("got %v, %v", got, err)
	}
	t.Setenv("PATH", t.TempDir())
	if _, err := List(); err == nil {
		t.Error("a missing fc-list should be an error")
	}
}
