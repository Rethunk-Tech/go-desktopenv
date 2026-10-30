//go:build !linux && !windows

package schemehandler

// Handler is a Config's registration, which this system does not support.
type Handler struct{}

// New returns a Handler whose every call is ErrUnsupported.
func New(Config) (*Handler, error) { return &Handler{}, nil }

// Owner returns ErrUnsupported.
func (*Handler) Owner(string) (Owner, error) { return Owner{}, ErrUnsupported }

// Register returns ErrUnsupported.
func (*Handler) Register(...string) error { return ErrUnsupported }

// Restore returns ErrUnsupported.
func (*Handler) Restore(map[string]string) error { return ErrUnsupported }

// Release returns ErrUnsupported.
func (*Handler) Release([]string, map[string]string) error { return ErrUnsupported }

// Forward returns ErrUnsupported.
func (*Handler) Forward(string, string) error { return ErrUnsupported }
