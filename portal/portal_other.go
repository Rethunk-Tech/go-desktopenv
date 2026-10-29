//go:build !linux

package portal

import "context"

// SetAutostart returns ErrUnsupported.
func SetAutostart(context.Context, Autostart) error { return ErrUnsupported }

// InstallLauncher returns ErrUnsupported.
func InstallLauncher(context.Context, Launcher) error { return ErrUnsupported }

// UninstallLauncher returns ErrUnsupported.
func UninstallLauncher(context.Context, string) error { return ErrUnsupported }

// LauncherExists returns ErrUnsupported.
func LauncherExists(context.Context, string) (bool, error) { return false, ErrUnsupported }

// RetrieveSecret returns ErrUnsupported.
func RetrieveSecret(context.Context) ([]byte, error) { return nil, ErrUnsupported }

// ReadSetting returns ErrUnsupported.
func ReadSetting(context.Context, string, string) (any, error) { return nil, ErrUnsupported }
