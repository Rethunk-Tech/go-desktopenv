// Package lnk writes and reads Windows .lnk shortcut files, the Windows counterpart of package desktopentry.
package lnk

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Shortcut is what a .lnk launches. Target is required; an empty field of the others is left unset.
type Shortcut struct {
	Target      string
	Arguments   string
	WorkingDir  string
	Icon        string // "path,index", as the shell's IconLocation reads it
	Description string
}

// Files lists the .lnk files directly in dir; a missing dir holds none.
func Files(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".lnk") {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	return paths, err
}
