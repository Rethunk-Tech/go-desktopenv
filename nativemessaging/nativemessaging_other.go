//go:build !linux && !windows

package nativemessaging

// Install returns ErrUnsupported.
func (Host) Install() error { return ErrUnsupported }

// Uninstall returns ErrUnsupported.
func (Host) Uninstall() error { return ErrUnsupported }

// Status returns ErrUnsupported.
func (Host) Status() ([]Status, error) { return nil, ErrUnsupported }
