package userdirs

import "strings"

// foldCase makes two spellings of one Windows path compare equal.
func foldCase(p string) string { return strings.ToLower(p) }
