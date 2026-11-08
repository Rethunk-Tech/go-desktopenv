//go:build windows

package wallpaper

import "golang.org/x/sys/windows/registry"

// Path returns the path of the user's desktop wallpaper, or empty when it cannot be read.
func Path() string {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Desktop`, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer func() { _ = k.Close() }()
	v, _, err := k.GetStringValue("Wallpaper")
	if err != nil {
		return ""
	}
	return v
}
