package sysaccent

import (
	"math"
	"testing"
)

func TestFromDWMReadsABGR(t *testing.T) {
	if got := FromDWM(0xFFB07A3C); got != "#3C7AB0" {
		t.Errorf("got %s", got)
	}
}

func TestFromPortal(t *testing.T) {
	for name, c := range map[string]struct {
		in   any
		want string
		ok   bool
	}{
		"rgb":          {[]any{1.0, 0.0, 0.5019607843137255}, "#FF0080", true},
		"black":        {[]any{0.0, 0.0, 0.0}, "#000000", true},
		"unset is -1":  {[]any{-1.0, -1.0, -1.0}, "", false},
		"wrong shape":  {[]any{0.1, 0.2}, "", false},
		"not a colour": {"red", "", false},
		"not a float":  {[]any{0.0, "x", 0.0}, "", false},
		"above one":    {[]any{1.1, 0.0, 0.0}, "", false},
		"NaN":          {[]any{math.NaN(), 0.0, 0.0}, "", false},
	} {
		if got, ok := FromPortal(c.in); got != c.want || ok != c.ok {
			t.Errorf("%s: %q %v", name, got, ok)
		}
	}
}

func TestReadReturnsWhetherTheDesktopSetsAnAccent(t *testing.T) {
	got, ok := Read()
	if ok {
		if len(got) != 7 || got[0] != '#' {
			t.Errorf("got %q", got)
		}
		return
	}
	if got != "" {
		t.Errorf("got %q", got)
	}
}
