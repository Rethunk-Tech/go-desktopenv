package sysaccent

import "testing"

func TestReadReportsNoAccentWhenThePortalIsAbsent(t *testing.T) {
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/no/such/desktopenv-bus")
	got, ok := Read()
	if got != "" || ok {
		t.Errorf("got %q %v", got, ok)
	}
}
