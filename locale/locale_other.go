//go:build !windows

package locale

import "os"

// Language is the tag of the first of LC_ALL, LC_MESSAGES and LANG that is set, as POSIX orders them; "" when none is
// set or the first is "C" or "POSIX".
func Language() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(key); v != "" {
			return FromPOSIX(v)
		}
	}
	return ""
}
