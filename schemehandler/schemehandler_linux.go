package schemehandler

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-desktopenv/desktopentry"
	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-desktopenv/internal/hostfile"
	"github.com/Rethunk-Tech/go-desktopenv/userdirs"
	"github.com/Rethunk-Tech/go-safefs"
)

const (
	xdgMime     = "/usr/bin/xdg-mime"
	updateDB    = "update-desktop-database"
	desktopPerm = 0o644
)

// Handler is one Config's registration on this system.
type Handler struct {
	cfg      Config
	home     string
	dataHome string
	// dataDirs are the system applications folders, read for an owner's name and Exec.
	dataDirs []string
	run      hostfile.Runner
}

// New returns the Handler for cfg. It reads HOME, XDG_DATA_HOME and XDG_DATA_DIRS once.
func New(cfg Config) (*Handler, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	h := &Handler{cfg: cfg, home: home, dataHome: userdirs.DataHome(home), run: hostfile.Exec}
	if cfg.Run != nil {
		h.run = cfg.Run
	}
	dirs := cmp.Or(os.Getenv("XDG_DATA_DIRS"), "/usr/local/share:/usr/share")
	for d := range strings.SplitSeq(dirs, ":") {
		h.dataDirs = append(h.dataDirs, filepath.Join(d, "applications"))
	}
	return h, nil
}

// packaged is Config.Packaged, or a Flatpak, whose own entry is the one its package exported.
func (h *Handler) packaged() bool { return h.cfg.Packaged || flatpak.Inside() }

func (h *Handler) host() hostfile.Host { return hostfile.Host{Run: h.run} }

// mimeDefaults runs xdg-mime. Inside a Flatpak the runtime has none, and the defaults that count are the host's.
func (h *Handler) mimeDefaults(args ...string) (string, error) {
	return h.host().Command(xdgMime, args...)
}

func schemeMime(scheme string) string { return "x-scheme-handler/" + scheme }

func (h *Handler) schemeMimes() []string {
	out := make([]string, 0, len(h.cfg.Schemes))
	for _, scheme := range h.cfg.Schemes {
		out = append(out, schemeMime(scheme))
	}
	return out
}

func (h *Handler) apps() string { return filepath.Join(h.dataHome, "applications") }

func (h *Handler) desktopPath() string { return filepath.Join(h.apps(), h.cfg.DesktopID) }

// SetDefault makes the handler's entry the default for each MIME type, such as the app's own file type or a scheme
// that is never handed back.
func (h *Handler) SetDefault(mimes ...string) error {
	for _, mime := range mimes {
		if _, err := h.mimeDefaults("default", h.handlerID(), mime); err != nil {
			return fmt.Errorf("xdg-mime default: %w", err)
		}
	}
	return nil
}

// Owner is the current owner of scheme.
func (h *Handler) Owner(scheme string) (Owner, error) {
	out, err := h.mimeDefaults("query", "default", schemeMime(scheme))
	if err != nil {
		return Owner{}, fmt.Errorf("xdg-mime query: %w", err)
	}
	id := strings.TrimSpace(out)
	return Owner{ID: id, Name: h.appName(id), Mine: id != "" && id == h.handlerID()}, nil
}

// appName is the Name= of the desktop file, or the file's id without its extension.
func (h *Handler) appName(id string) string {
	if name, err := h.entryKey(id, "Name="); err == nil {
		return name
	}
	return strings.TrimSuffix(id, ".desktop")
}

func readKey(path, prefix string) string {
	b, err := safefs.ReadFile(path)
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, prefix); ok {
			return v
		}
	}
	return ""
}

// entryKey is the first value of a key in the entry id, from the user's applications folder or a system one.
func (h *Handler) entryKey(id, prefix string) (string, error) {
	for _, dir := range append([]string{h.apps()}, h.dataDirs...) {
		if v := readKey(filepath.Join(dir, id), prefix); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("desktop entry %q not found", id)
}

// desktopFile is the user entry. It lists the schemes only while they are claimed, so a system that picks a default
// from advertised types never picks this program once the user has switched it off.
func (h *Handler) desktopFile(withSchemes bool) string {
	var mime []string
	if withSchemes {
		mime = h.schemeMimes()
	}
	mime = append(mime, h.cfg.Entry.MimeTypes...)
	e := h.cfg.Entry
	var b strings.Builder
	b.WriteString("[Desktop Entry]\nType=Application\nName=" + desktopentry.Value(e.Name) + "\n")
	if e.Comment != "" {
		b.WriteString("Comment=" + desktopentry.Value(e.Comment) + "\n")
	}
	b.WriteString("Exec=" + desktopentry.Exec(h.cfg.Exe) + " %u\n")
	if e.Icon != "" {
		b.WriteString("Icon=" + desktopentry.Value(e.Icon) + "\n")
	}
	b.WriteString("Terminal=false\n")
	writeList(&b, "Categories", e.Categories)
	writeList(&b, "Keywords", e.Keywords)
	if e.StartupWMClass != "" {
		b.WriteString("StartupWMClass=" + desktopentry.Value(e.StartupWMClass) + "\n")
	}
	writeList(&b, "MimeType", mime)
	return b.String()
}

// writeList writes a list key; a `;` inside an item is escaped so it cannot add an item.
func writeList(b *strings.Builder, key string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString(key + "=")
	for _, it := range items {
		b.WriteString(strings.ReplaceAll(desktopentry.Value(it), ";", `\;`) + ";")
	}
	b.WriteString("\n")
}

// integrator is the id of an entry in the user's applications folder that an AppImage integrator (GearLever,
// AppImageLauncher, appimaged) wrote to run Exe on a URL, or "". That entry is then the launcher and the handler,
// and an entry of the app's own beside it would list the app twice for every scheme. An entry whose Exec sets
// DESKTOPINTEGRATION, the integrators' mark, wins over one that only names the program.
func (h *Handler) integrator() string {
	if h.packaged() {
		return ""
	}
	entries, err := os.ReadDir(h.apps())
	if err != nil {
		return ""
	}
	own := strings.TrimSuffix(h.cfg.DesktopID, ".desktop")
	found := ""
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".desktop") || strings.HasPrefix(name, own) {
			continue
		}
		argv := desktopentry.SplitExec(readKey(filepath.Join(h.apps(), name), "Exec="))
		marked := false
		if len(argv) > 0 && filepath.Base(argv[0]) == "env" {
			argv = argv[1:]
			for len(argv) > 0 && strings.Contains(argv[0], "=") {
				marked = marked || strings.HasPrefix(argv[0], "DESKTOPINTEGRATION=")
				argv = argv[1:]
			}
		}
		if len(argv) < 2 || argv[0] != h.cfg.Exe || !slices.ContainsFunc(argv[1:], func(a string) bool { return a == "%u" || a == "%U" }) {
			continue
		}
		if marked {
			return name
		}
		found = cmp.Or(found, name)
	}
	return found
}

// handlerID is the entry made the default: an integrator's when there is one, else the app's own.
func (h *Handler) handlerID() string { return cmp.Or(h.integrator(), h.cfg.DesktopID) }

// yieldTo removes the app's own entry in favour of the integrator's entry id and moves every default it held there.
func (h *Handler) yieldTo(id string) error {
	path := h.desktopPath()
	if _, err := safefs.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err := safefs.RemoveAll(path); err != nil {
		return err
	}
	_, _ = h.run(updateDB, filepath.Dir(path))
	for _, mime := range append(h.schemeMimes(), h.cfg.Entry.MimeTypes...) {
		if out, err := h.mimeDefaults("query", "default", mime); err == nil && strings.TrimSpace(out) == h.cfg.DesktopID {
			if _, err := h.mimeDefaults("default", id, mime); err != nil {
				return fmt.Errorf("xdg-mime default: %w", err)
			}
		}
	}
	return nil
}

func (h *Handler) writeDesktop(withSchemes bool) error {
	if h.packaged() {
		return nil
	}
	if id := h.integrator(); id != "" {
		return h.yieldTo(id)
	}
	path := h.desktopPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	if err := safefs.WriteFile(path, []byte(h.desktopFile(withSchemes)), desktopPerm); err != nil {
		return err
	}
	// A missing tool or a failure only leaves the cache stale; xdg-mime reads the file itself.
	_, _ = h.run(updateDB, filepath.Dir(path))
	return nil
}

// Register writes the user entry listing every scheme, or yields to an AppImage integrator's entry for Exe, then
// makes that entry the default for each of schemes.
func (h *Handler) Register(schemes ...string) error {
	if err := h.writeDesktop(true); err != nil {
		return err
	}
	for _, scheme := range schemes {
		if err := h.SetDefault(schemeMime(scheme)); err != nil {
			return err
		}
	}
	return nil
}

// Restore rewrites the user entry without the schemes and hands every Config scheme back to its ID in previous, or
// drops the default when it has none.
func (h *Handler) Restore(previous map[string]string) error {
	if err := h.writeDesktop(false); err != nil {
		return err
	}
	return h.handBack(h.cfg.Schemes, previous)
}

// Release hands back only schemes, as Restore does, and leaves the rest claimed; the entry still lists every scheme.
func (h *Handler) Release(schemes []string, previous map[string]string) error {
	if err := h.writeDesktop(true); err != nil {
		return err
	}
	return h.handBack(schemes, previous)
}

func (h *Handler) handBack(schemes []string, previous map[string]string) error {
	var unowned []string
	for _, scheme := range schemes {
		if previous[scheme] == "" {
			unowned = append(unowned, schemeMime(scheme))
			continue
		}
		if _, err := h.mimeDefaults("default", previous[scheme], schemeMime(scheme)); err != nil {
			return fmt.Errorf("xdg-mime default: %w", err)
		}
	}
	if len(unowned) == 0 {
		return nil
	}
	return h.DropDefault(unowned...)
}

// launchOnHost starts the desktop entry id on a URL with the host's gio, finding the entry where the host's XDG
// data folders put it; the sandbox sees neither those folders nor the program.
const launchOnHost = `dirs="${XDG_DATA_HOME:-$HOME/.local/share}:${XDG_DATA_DIRS:-/usr/local/share:/usr/share}"
IFS=:
for d in $dirs; do
	if [ -f "$d/applications/$1" ]; then exec gio launch "$d/applications/$1" "$2"; fi
done
echo "no desktop entry $1" >&2
exit 1`

// Forward runs the desktop entry previous (an Owner.ID) on link, which stays one argument. Inside a Flatpak the
// host's gio launches it.
func (h *Handler) Forward(link, previous string) error {
	if flatpak.Inside() {
		if previous == "" {
			return errNoPrevious
		}
		_, err := h.host().Command("sh", "-c", launchOnHost, "sh", previous, link)
		return err
	}
	name, args, err := linuxForwardArgv(previous, link, func(id string) (string, error) {
		return h.entryKey(id, "Exec=")
	})
	if err != nil {
		return err
	}
	_, err = h.run(name, args...)
	return err
}

// DropDefault removes the lines that make the handler's entry the default for mimes from the user's mimeapps.list,
// following a symlink to its target: xdg-mime cannot unset a default.
func (h *Handler) DropDefault(mimes ...string) error {
	host := h.host()
	path := host.Resolve(filepath.Join(hostfile.ConfigHome(h.home), "mimeapps.list"))
	b, err := host.Read(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	ids := []string{h.cfg.DesktopID, h.handlerID()}
	kept := slices.DeleteFunc(slices.Clone(lines), func(s string) bool {
		k, v, ok := strings.Cut(strings.TrimSpace(s), "=")
		return ok && slices.Contains(mimes, k) && slices.Contains(ids, strings.TrimSuffix(v, ";"))
	})
	if len(kept) == len(lines) {
		return nil
	}
	return host.Write(path, []byte(strings.Join(kept, "\n")), desktopPerm)
}

// WriteEntry writes the user entry unless it already reads as it should, keeping whether it lists the schemes, or
// yields to an integrator's entry. A Packaged Config has no user entry.
func (h *Handler) WriteEntry() error {
	if h.packaged() {
		return nil
	}
	if id := h.integrator(); id != "" {
		return h.yieldTo(id)
	}
	current, _ := safefs.ReadFile(h.desktopPath())
	mimes := h.schemeMimes()
	withSchemes := len(mimes) > 0 && strings.Contains(string(current), mimes[0])
	if string(current) == h.desktopFile(withSchemes) {
		return nil
	}
	return h.writeDesktop(withSchemes)
}

// EntryProgram is the program the user entry runs, read back from the quoted Exec this package writes, or "" when
// the entry holds none. It is fs.ErrNotExist when there is no user entry, which is how a caller tells that another
// copy of the app (a second AppImage) owns the entry: the program differs from Exe and still exists.
func (h *Handler) EntryProgram() (string, error) {
	b, err := safefs.ReadFile(h.desktopPath())
	if err != nil {
		return "", err
	}
	return execTarget(b), nil
}

func execTarget(entry []byte) string {
	for line := range strings.SplitSeq(string(entry), "\n") {
		if rest, ok := strings.CutPrefix(line, "Exec="); ok && strings.HasPrefix(rest, `"`) {
			return desktopentry.SplitExec(rest)[0]
		}
	}
	return ""
}

// RemoveEntry deletes the user entry, for an uninstall. Call Restore first to hand the schemes back.
func (h *Handler) RemoveEntry() error {
	path := h.desktopPath()
	if _, err := safefs.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err := safefs.RemoveAll(path); err != nil {
		return err
	}
	_, _ = h.run(updateDB, filepath.Dir(path))
	return nil
}
