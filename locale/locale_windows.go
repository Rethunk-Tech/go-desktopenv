package locale

import "golang.org/x/sys/windows"

const muiLanguageName = 0x8

// Language is the user's preferred display language as a tag such as "de-DE"; "" when Windows names none.
func Language() string {
	langs, err := windows.GetUserPreferredUILanguages(muiLanguageName)
	if err != nil || len(langs) == 0 {
		return ""
	}
	return langs[0]
}
