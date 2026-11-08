package wallpaper

import (
	"testing"

	"golang.org/x/sys/windows/registry"
)

func TestPathReadsDesktopWallpaper(t *testing.T) {
	want := ""
	k, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\Desktop`, registry.QUERY_VALUE)
	if err == nil {
		defer func() { _ = k.Close() }()
		if v, _, err := k.GetStringValue("Wallpaper"); err == nil {
			want = v
		}
	}
	if got := Path(); got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}
