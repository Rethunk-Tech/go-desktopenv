//go:build !windows

package locale

import "testing"

func TestLanguageFollowsPOSIXOrder(t *testing.T) {
	for name, c := range map[string]struct{ all, msgs, lang, want string }{
		"LC_ALL wins":      {"de_DE.UTF-8", "fr_FR", "en_US", "de-DE"},
		"then LC_MESSAGES": {"", "fr_FR.UTF-8", "en_US", "fr-FR"},
		"then LANG":        {"", "", "pt_BR.UTF-8", "pt-BR"},
		"none set":         {"", "", "", ""},
		"C is no language": {"C", "", "en_US", ""},
	} {
		t.Setenv("LC_ALL", c.all)
		t.Setenv("LC_MESSAGES", c.msgs)
		t.Setenv("LANG", c.lang)
		if got := Language(); got != c.want {
			t.Errorf("%s: got %q, want %q", name, got, c.want)
		}
	}
}
