// Package locale reads the language the user has chosen for their desktop.
package locale

import "strings"

// FromPOSIX turns a POSIX locale such as "de_DE.UTF-8@euro" into the tag "de-DE"; "C" and "POSIX" name no language,
// so they give "".
func FromPOSIX(v string) string {
	if i := strings.IndexAny(v, ".@"); i >= 0 {
		v = v[:i]
	}
	if v == "C" || v == "POSIX" {
		return ""
	}
	return strings.ReplaceAll(v, "_", "-")
}
