package locale

import "testing"

func TestFromPOSIX(t *testing.T) {
	for in, want := range map[string]string{
		"de_DE.UTF-8@euro": "de-DE",
		"en_US.UTF-8":      "en-US",
		"fr":               "fr",
		"C":                "",
		"POSIX":            "",
		"C.UTF-8":          "",
		"":                 "",
	} {
		if got := FromPOSIX(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}
