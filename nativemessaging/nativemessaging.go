// Package nativemessaging lets a browser extension start this program as a native-messaging host: it writes and
// removes the host manifests Chromium-family browsers and Firefox look for, reports whether each installed browser's
// manifest points at the program, and reads and writes the stdio frames the browser exchanges with the host.
package nativemessaging

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrUnsupported is returned by Install, Uninstall and Status on a system that is neither Linux nor Windows.
var ErrUnsupported = errors.New("nativemessaging: host manifests are not supported on this system")

// Host is one native-messaging host. On Linux its manifests go to every installed browser's own folder, through the
// host from inside a Flatpak; on Windows to ManifestDir, with an HKCU key per browser pointing at them.
type Host struct {
	// Name is the host name extensions connect to, such as "org.example.app".
	Name string
	// Description is the manifest's description.
	Description string
	// Path is the program the browser starts. Inside a Flatpak, Install writes a wrapper that runs Path's base name
	// with flatpak run, and the manifests name that wrapper instead.
	Path string
	// AllowedOrigins are the Chromium-family extension origins allowed to connect, "chrome-extension://<id>/".
	AllowedOrigins []string
	// AllowedExtensions are the Firefox extension ids allowed to connect.
	AllowedExtensions []string
	// ManifestDir is the folder Windows keeps the two manifest files in. Unused on Linux.
	ManifestDir string
	// Run replaces the program runner on Linux; nil runs the real program. It gets every program the host starts
	// (inside a Flatpak, "flatpak-spawn" with "--host" first and the script's arguments after) as name and args
	// exactly as exec would take them, and returns the program's standard output. A non-nil error is a failed run.
	// Windows ignores it.
	Run func(name string, args ...string) (string, error)
}

// Manifest is the host manifest a browser reads; firefox picks that browser's allow-list field.
func (h Host) Manifest(firefox bool) ([]byte, error) { return h.manifest(h.Path, firefox) }

func (h Host) manifest(path string, firefox bool) ([]byte, error) {
	m := map[string]any{
		"name":        h.Name,
		"description": h.Description,
		"path":        path,
		"type":        "stdio",
	}
	if firefox {
		m["allowed_extensions"] = h.AllowedExtensions
	} else {
		m["allowed_origins"] = h.AllowedOrigins
	}
	return json.MarshalIndent(m, "", "  ")
}

// State is how one browser's manifest stands.
type State string

const (
	// OK is a manifest that names the program.
	OK State = "ok"
	// Missing is an installed browser with no manifest.
	Missing State = "missing"
	// Stale is a manifest that names another program, such as a copy that has moved.
	Stale State = "stale"
	// Unreadable is a manifest, or on Windows a registry key, that cannot be read or holds no path.
	Unreadable State = "unreadable"
)

// Status is one installed browser's manifest. Manifest is the file, or on Windows the registry key when it has no
// file.
type Status struct {
	Browser  string
	Manifest string
	State    State
}

// stateOf is the State of a manifest already read, with the read's error, for a host that should run exe.
func stateOf(exe string, b []byte, err error) State {
	if errors.Is(err, fs.ErrNotExist) {
		return Missing
	}
	if err != nil {
		return Unreadable
	}
	var m struct {
		Path string `json:"path"`
	}
	if json.Unmarshal(b, &m) != nil || m.Path == "" {
		return Unreadable
	}
	want, got := filepath.Clean(exe), filepath.Clean(m.Path)
	if want == got || (runtime.GOOS == "windows" && strings.EqualFold(want, got)) {
		return OK
	}
	return Stale
}

// Read reads one message from the browser into v: a 4-byte length in native byte order, then that many bytes of
// JSON. A message longer than limit is an error before any of it is read. A stream that ends cleanly between
// messages gives io.EOF.
func Read(r io.Reader, limit int, v any) error {
	var n uint32
	if err := binary.Read(r, binary.NativeEndian, &n); err != nil {
		return err
	}
	if uint64(n) > uint64(max(limit, 0)) {
		return fmt.Errorf("nativemessaging: message of %d bytes is over the %d-byte limit", n, limit)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		if errors.Is(err, io.EOF) {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	return json.Unmarshal(b, v)
}

// Write writes v to the browser as one message, its JSON after the length prefix, in a single Write.
func Write(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	n := len(b)
	if n < 0 || uint64(n) > math.MaxUint32 {
		return fmt.Errorf("nativemessaging: message of %d bytes cannot be framed", n)
	}
	frame := binary.NativeEndian.AppendUint32(make([]byte, 0, 4+n), uint32(n))
	_, err = w.Write(append(frame, b...))
	return err
}
