package locale

import (
	"strings"
	"testing"
)

// The windows-latest leg is the only place this reads a real user setting; it can only check the form.
func TestLanguageIsATag(t *testing.T) {
	if got := Language(); strings.ContainsAny(got, "_. ") {
		t.Errorf("not a BCP-47 tag: %q", got)
	}
}
