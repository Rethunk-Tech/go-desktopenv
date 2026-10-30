// Package fonts lists the font families installed on this computer.
package fonts

import (
	"slices"
	"strings"
)

// ParseFcList reads `fc-list : family` output: one line per font file, each holding the family's names separated by
// commas, of which the first is the family. The result is sorted and has each family once.
func ParseFcList(out string) []string {
	var families []string
	for line := range strings.SplitSeq(out, "\n") {
		first, _, _ := strings.Cut(line, ",")
		families = append(families, strings.TrimSpace(first))
	}
	return tidy(families)
}

var styleWords = []string{
	"Bold", "Italic", "Oblique", "Regular", "Light", "Semilight", "Semibold", "Medium", "Black", "Thin", "Heavy",
	"ExtraBold", "ExtraLight", "UltraLight", "Demi", "Book",
}

// ParseWindowsNames reads the value names of Windows' installed fonts key ("Arial Bold (TrueType)", "Segoe UI Bold &
// Segoe UI Bold Italic (TrueType)") as family names, without the style words.
func ParseWindowsNames(names []string) []string {
	var families []string
	for _, n := range names {
		if i := strings.LastIndex(n, " ("); i >= 0 {
			n = n[:i]
		}
		for part := range strings.SplitSeq(n, " & ") {
			words := strings.Fields(part)
			for len(words) > 1 && slices.Contains(styleWords, words[len(words)-1]) {
				words = words[:len(words)-1]
			}
			families = append(families, strings.Join(words, " "))
		}
	}
	return tidy(families)
}

func tidy(families []string) []string {
	families = slices.DeleteFunc(families, func(f string) bool { return f == "" })
	slices.SortFunc(families, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) })
	return slices.CompactFunc(families, strings.EqualFold)
}
