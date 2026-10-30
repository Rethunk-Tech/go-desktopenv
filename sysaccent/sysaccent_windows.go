package sysaccent

import "golang.org/x/sys/windows/registry"

// Read is the DWM accent colour as "#rrggbb"; false when the value is absent.
func Read() (string, bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\DWM`, registry.QUERY_VALUE)
	if err != nil {
		return "", false
	}
	defer func() { _ = k.Close() }()
	v, _, err := k.GetIntegerValue("AccentColor")
	if err != nil {
		return "", false
	}
	return FromDWM(v), true
}
