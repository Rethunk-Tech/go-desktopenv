package schemehandler

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/nowindow"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Handler is one Config's registration on this system.
type Handler struct {
	cfg Config
	// classes is the HKCU key the scheme keys live under.
	classes string
}

// New returns the Handler for cfg.
func New(cfg Config) (*Handler, error) { return &Handler{cfg: cfg, classes: `Software\Classes`}, nil }

func (h *Handler) classKey(scheme string) string   { return h.classes + `\` + scheme }
func (h *Handler) commandKey(scheme string) string { return h.classKey(scheme) + `\shell\open\command` }
func (h *Handler) iconKey(scheme string) string    { return h.classKey(scheme) + `\DefaultIcon` }

func (h *Handler) command() string { return `"` + h.cfg.Exe + `" "%1"` }

// Owner is the current owner of scheme. Its ID saves the open command, icon and name, for Restore.
func (h *Handler) Owner(scheme string) (Owner, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, h.commandKey(scheme), registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return Owner{}, nil
	}
	if err != nil {
		return Owner{}, err
	}
	defer func() { _ = k.Close() }()
	cmd, _, err := k.GetStringValue("")
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		return Owner{}, err
	}
	if cmd == "" {
		return Owner{}, nil
	}
	id := previousID(cmd, readDefault(h.iconKey(scheme)), readDefault(h.classKey(scheme)))
	return Owner{ID: id, Name: exeOf(cmd), Mine: sameExe(exeOf(cmd), h.cfg.Exe)}, nil
}

// sameExe compares program paths the way Windows does: case-insensitively, with 8.3 short names expanded.
func sameExe(a, b string) bool { return strings.EqualFold(longPath(a), longPath(b)) }

func longPath(p string) string {
	in, err := windows.UTF16PtrFromString(p)
	if err != nil {
		return p
	}
	n, err := windows.GetLongPathName(in, nil, 0)
	if err != nil || n == 0 {
		return p
	}
	buf := make([]uint16, n)
	n, err = windows.GetLongPathName(in, &buf[0], n)
	if err != nil || n == 0 || int(n) > len(buf) {
		return p
	}
	return windows.UTF16ToString(buf[:n])
}

// exeOf is the program of an open command.
func exeOf(cmd string) string {
	if rest, ok := strings.CutPrefix(cmd, `"`); ok {
		exe, _, _ := strings.Cut(rest, `"`)
		return exe
	}
	exe, _, _ := strings.Cut(cmd, " ")
	return exe
}

// Register writes each scheme's keys: the URL protocol marker, Exe's icon and `"Exe" "%1"` as the open command.
func (h *Handler) Register(schemes ...string) error {
	for _, scheme := range schemes {
		if err := h.registerScheme(scheme); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) registerScheme(scheme string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, h.classKey(scheme), registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	if err := k.SetStringValue("", "URL:"+strings.ToUpper(scheme)+" Protocol"); err != nil {
		return err
	}
	if err := k.SetStringValue("URL Protocol", ""); err != nil {
		return err
	}
	if err := setDefault(h.iconKey(scheme), h.cfg.Exe+",0"); err != nil {
		return err
	}
	return setDefault(h.commandKey(scheme), h.command())
}

func setDefault(key, value string) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, key, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func() { _ = k.Close() }()
	return k.SetStringValue("", value)
}

// readDefault is the key's default value, or "" when it has none.
func readDefault(key string) string {
	k, err := registry.OpenKey(registry.CURRENT_USER, key, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer func() { _ = k.Close() }()
	v, _, err := k.GetStringValue("")
	if err != nil {
		return ""
	}
	return v
}

// Restore hands every Config scheme back to its ID in previous; a scheme with none has its keys deleted. A scheme
// another program owns is left alone. Every scheme is tried and the errors are joined.
func (h *Handler) Restore(previous map[string]string) error {
	return h.Release(h.cfg.Schemes, previous)
}

// Release hands back only schemes, as Restore does.
func (h *Handler) Release(schemes []string, previous map[string]string) error {
	var errs []error
	for _, scheme := range schemes {
		errs = append(errs, h.restoreScheme(scheme, previous[scheme]))
	}
	return errors.Join(errs...)
}

// restoreScheme acts only on a scheme Exe holds: another program's registration is not this one's to delete.
func (h *Handler) restoreScheme(scheme, saved string) error {
	if o, err := h.Owner(scheme); err != nil || !o.Mine {
		return err
	}
	if saved == "" {
		class := h.classKey(scheme)
		for _, key := range []string{h.iconKey(scheme), h.commandKey(scheme), class + `\shell\open`, class + `\shell`, class} {
			if err := registry.DeleteKey(registry.CURRENT_USER, key); err != nil && !errors.Is(err, registry.ErrNotExist) {
				return fmt.Errorf("delete %s: %w", key, err)
			}
		}
		return nil
	}
	p := splitPrevious(saved)
	if err := setDefault(h.commandKey(scheme), p.cmd); err != nil {
		return err
	}
	if p.hasName {
		if err := setDefault(h.classKey(scheme), p.name); err != nil {
			return err
		}
	}
	if !p.hasIcon {
		return nil
	}
	if p.icon == "" {
		if err := registry.DeleteKey(registry.CURRENT_USER, h.iconKey(scheme)); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}
	return setDefault(h.iconKey(scheme), p.icon)
}

// Forward starts the saved open command previous (an Owner.ID) with link in place of %1, as one argument, without
// a console window, and does not wait for it: the previous handler may run until the user closes it.
func (h *Handler) Forward(link, previous string) error {
	name, args, err := windowsForwardArgv(previous, link)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(context.Background(), name, args...) //nolint:gosec // G204: name is the handler the user had registered
	nowindow.Set(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
