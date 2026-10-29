package userdirs

import (
	"strings"
	"testing"
)

func TestARealHomeSpelledInAnotherCaseStillCounts(t *testing.T) {
	home := realHome(t)
	custom := t.TempDir()
	t.Setenv("APPDATA", custom)
	if got := AppData(strings.ToUpper(home)); got != custom {
		t.Fatalf("%q, want %q", got, custom)
	}
}
