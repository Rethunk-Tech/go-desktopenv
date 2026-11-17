package sysaccent

import (
	"math"
	"testing"
)

func FuzzFromPortal(f *testing.F) {
	f.Add(1.0, 0.0, 0.5019607843137255)
	f.Add(0.0, 0.0, 0.0)
	f.Add(-1.0, -1.0, -1.0)
	f.Add(1.1, 0.0, 0.0)
	f.Add(math.NaN(), 0.0, 0.0)
	f.Fuzz(func(t *testing.T, r, g, b float64) {
		got, ok := FromPortal([]any{r, g, b})
		if !ok {
			if got != "" {
				t.Fatalf("FromPortal(%v, %v, %v) = %q, false", r, g, b, got)
			}
			return
		}
		if len(got) != 7 || got[0] != '#' {
			t.Fatalf("FromPortal(%v, %v, %v) = %q", r, g, b, got)
		}
		for _, c := range got[1:] {
			if c >= '0' && c <= '9' || c >= 'A' && c <= 'F' {
				continue
			}
			t.Fatalf("FromPortal(%v, %v, %v) = %q is not #RRGGBB", r, g, b, got)
		}
	})
}
