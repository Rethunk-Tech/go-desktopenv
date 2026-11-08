//go:build !windows

package firewall

// CurrentProfileTypes returns the connected networks' Windows Firewall profile
// bits (NET_FW_PROFILE2_*). It returns ErrUnsupported on this operating system.
func CurrentProfileTypes() (int32, error) {
	return 0, ErrUnsupported
}

// Rules returns the inbound-relevant fields of every Windows Firewall rule. It
// returns ErrUnsupported on this operating system.
func Rules() ([]Rule, error) {
	return nil, ErrUnsupported
}

// Add removes inbound block rules for exe and writes an inbound allow rule. It
// returns ErrUnsupported on this operating system.
func Add(exe, name, publicName string, anyAddr bool) error {
	return ErrUnsupported
}

// Elevate starts exe with args via ShellExecuteExW runas and waits for its
// exit code. It returns ErrUnsupported on this operating system.
func Elevate(exe, args string) error {
	return ErrUnsupported
}
