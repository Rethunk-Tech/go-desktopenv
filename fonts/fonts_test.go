package fonts

import (
	"slices"
	"testing"
)

func TestParseFcListKeepsOneFamilyEach(t *testing.T) {
	got := ParseFcList("DejaVu Sans,DejaVu Sans Condensed\nOpen Sans\ndejavu sans\n\nNoto Serif,Noto Serif CJK\n")
	if want := []string{"DejaVu Sans", "Noto Serif", "Open Sans"}; !slices.Equal(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestParseWindowsNamesDropsStyles(t *testing.T) {
	got := ParseWindowsNames([]string{"Arial (TrueType)", "Arial Bold Italic (TrueType)", "Segoe UI Light & Segoe UI Semilight (TrueType)", "Segoe UI (TrueType)", "Black (OpenType)"})
	if want := []string{"Arial", "Black", "Segoe UI"}; !slices.Equal(got, want) {
		t.Errorf("got %v", got)
	}
}
