package portal

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/Rethunk-Tech/go-desktopenv/desktopentry"
)

const (
	busName    = "org.freedesktop.portal.Desktop"
	objectPath = "/org/freedesktop/portal/desktop"
	launcherIf = "org.freedesktop.portal.DynamicLauncher"
	requestIf  = "org.freedesktop.portal.Request"
	notFoundIf = "org.freedesktop.portal.Error.NotFound"
)

// closeGrace bounds the Request.Close sent after the caller's context ended, which can no longer bound it.
const closeGrace = 2 * time.Second

// newToken is a fresh handle token for one call. Each call already has its own connection, and so its own Request
// path; a random token keeps the path from naming the app.
func newToken(kind string) string {
	return "go_desktopenv_" + kind + "_" + strings.ToLower(rand.Text())
}

// handlePath is the Request object the portal creates for a caller's unique bus name and handle token.
func handlePath(uniqueName, token string) dbus.ObjectPath {
	sender := strings.ReplaceAll(strings.TrimPrefix(uniqueName, ":"), ".", "_")
	return dbus.ObjectPath(objectPath + "/request/" + sender + "/" + token)
}

func call(ctx context.Context, conn *dbus.Conn, method string, args ...any) *dbus.Call {
	return conn.Object(busName, objectPath).CallWithContext(ctx, method, 0, args...)
}

// request calls a portal method that answers through a Request object, and returns the Response's results. It
// subscribes to the Response before calling, since the portal may answer before the call returns. When ctx ends first
// the Request is closed, so the portal drops its dialog, and ctx.Err() is returned.
func request(ctx context.Context, conn *dbus.Conn, method, token string, args ...any) (map[string]dbus.Variant, error) {
	handle := handlePath(conn.Names()[0], token)
	match := []dbus.MatchOption{
		dbus.WithMatchObjectPath(handle),
		dbus.WithMatchInterface(requestIf),
		dbus.WithMatchMember("Response"),
	}
	if err := conn.AddMatchSignal(match...); err != nil {
		return nil, err
	}
	defer func() { _ = conn.RemoveMatchSignal(match...) }()
	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	if err := call(ctx, conn, method, args...).Err; err != nil {
		if ctx.Err() != nil {
			closeRequest(ctx, conn, handle)
			return nil, ctx.Err()
		}
		return nil, err
	}
	for {
		var sig *dbus.Signal
		select {
		case <-ctx.Done():
			closeRequest(ctx, conn, handle)
			return nil, ctx.Err()
		case s, ok := <-signals:
			if !ok {
				return nil, errors.New("the portal connection closed")
			}
			sig = s
		}
		if sig.Path != handle || len(sig.Body) < 2 {
			continue
		}
		code, _ := sig.Body[0].(uint32)
		results, _ := sig.Body[1].(map[string]dbus.Variant)
		switch code {
		case 0:
			return results, nil
		case 1:
			return nil, ErrCancelled
		default:
			return nil, fmt.Errorf("%s failed", method)
		}
	}
}

func closeRequest(ctx context.Context, conn *dbus.Conn, handle dbus.ObjectPath) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), closeGrace)
	defer cancel()
	_ = conn.Object(busName, handle).CallWithContext(ctx, requestIf+".Close", 0).Err
}

// SetAutostart asks the Background portal to start a.Command (run inside the sandbox) at sign-in, or to stop doing
// so. It returns ErrNotAllowed when enabling was not granted.
func SetAutostart(ctx context.Context, a Autostart) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	token := newToken("autostart")
	options := map[string]dbus.Variant{
		"handle_token":     dbus.MakeVariant(token),
		"autostart":        dbus.MakeVariant(a.Enable),
		"commandline":      dbus.MakeVariant(a.Command),
		"dbus-activatable": dbus.MakeVariant(false),
	}
	if a.Reason != "" {
		options["reason"] = dbus.MakeVariant(a.Reason)
	}
	results, err := request(ctx, conn, "org.freedesktop.portal.Background.RequestBackground", token, "", options)
	if err != nil {
		return err
	}
	if granted, _ := results["autostart"].Value().(bool); a.Enable && !granted {
		return ErrNotAllowed
	}
	return nil
}

// InstallLauncher asks the user, through the DynamicLauncher portal, to add the launcher l.
func InstallLauncher(ctx context.Context, l Launcher) error {
	if len(l.Exec) == 0 {
		return errors.New("portal: launcher has no command")
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	// A serialized GBytesIcon: ("bytes", <PNG data>).
	icon := dbus.MakeVariant(struct {
		Kind string
		Data dbus.Variant
	}{"bytes", dbus.MakeVariant(l.IconPNG)})
	handleToken := newToken("launcher")
	results, err := request(ctx, conn, launcherIf+".PrepareInstall", handleToken, "", l.Name, icon,
		map[string]dbus.Variant{
			"handle_token":  dbus.MakeVariant(handleToken),
			"launcher_type": dbus.MakeVariant(uint32(1)),
		})
	if err != nil {
		return err
	}
	token, _ := results["token"].Value().(string)
	if token == "" {
		return errors.New("the launcher portal returned no install token")
	}
	if chosen, _ := results["name"].Value().(string); chosen != "" {
		l.Name = chosen
	}
	return call(ctx, conn, launcherIf+".Install", token, l.ID, desktopEntry(l),
		map[string]dbus.Variant{}).Err
}

// desktopEntry is the launcher's Desktop Entry file. The portal rewrites Exec to run the launcher inside the sandbox.
func desktopEntry(l Launcher) string {
	var b strings.Builder
	b.WriteString("[Desktop Entry]\nType=Application\nName=" + desktopentry.Value(l.Name) + "\n")
	if l.Comment != "" {
		b.WriteString("Comment=" + desktopentry.Value(l.Comment) + "\n")
	}
	b.WriteString("Exec=" + desktopentry.Exec(l.Exec...) + "\nTerminal=false\n")
	if len(l.Categories) > 0 {
		b.WriteString("Categories=")
		for _, c := range l.Categories {
			// A semicolon separates list values, so one inside a value is escaped.
			b.WriteString(strings.ReplaceAll(desktopentry.Value(c), ";", `\;`) + ";")
		}
		b.WriteString("\n")
	}
	return b.String()
}

// UninstallLauncher removes a launcher InstallLauncher added; a missing one is not an error.
func UninstallLauncher(ctx context.Context, id string) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	if exists, err := launcherExists(ctx, conn, id); err != nil || !exists {
		return err
	}
	return call(ctx, conn, launcherIf+".Uninstall", id, map[string]dbus.Variant{}).Err
}

// LauncherExists reports whether InstallLauncher added the launcher id.
func LauncherExists(ctx context.Context, id string) (bool, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return false, err
	}
	defer func() { _ = conn.Close() }()
	return launcherExists(ctx, conn, id)
}

func launcherExists(ctx context.Context, conn *dbus.Conn, id string) (bool, error) {
	var entry string
	err := call(ctx, conn, launcherIf+".GetDesktopEntry", id).Store(&entry)
	if err == nil {
		return true, nil
	}
	// The portal reads the launcher from its own folder and reports a missing one as GLib's G_FILE_ERROR_NOENT (4).
	var dbusErr dbus.Error
	if errors.As(err, &dbusErr) && strings.HasSuffix(dbusErr.Name, "_g_2dfile_2derror_2dquark.Code4") {
		return false, nil
	}
	return false, err
}

// RetrieveSecret returns the Secret portal's per-app master secret: the same bytes on every call for this app id,
// which the portal derives from the app id, not from the request. It answers only a sandboxed caller with an app id.
func RetrieveSecret(ctx context.Context) ([]byte, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	fd := w.Fd()
	if fd > math.MaxInt32 {
		_ = w.Close()
		return nil, errors.New("pipe descriptor out of range")
	}
	token := newToken("secret")
	_, err = request(ctx, conn, "org.freedesktop.portal.Secret.RetrieveSecret", token,
		dbus.UnixFD(fd), map[string]dbus.Variant{"handle_token": dbus.MakeVariant(token)})
	// Our copy of the write end must close or the read never sees EOF.
	_ = w.Close()
	if err != nil {
		return nil, err
	}
	secret, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if len(secret) == 0 {
		return nil, errors.New("the secret portal returned an empty secret")
	}
	return secret, nil
}

// ReadSetting returns the Settings portal value for namespace/key with every enclosing variant removed (older portals
// wrap the value twice), or ErrNotFound when the portal has no such setting. For example
// ("org.freedesktop.appearance", "accent-color") answers the desktop's accent colour as []any of three float64.
func ReadSetting(ctx context.Context, namespace, key string) (any, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	var out dbus.Variant
	err = call(ctx, conn, "org.freedesktop.portal.Settings.Read", namespace, key).Store(&out)
	var dbusErr dbus.Error
	if errors.As(err, &dbusErr) && dbusErr.Name == notFoundIf {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return unwrap(out), nil
}

func unwrap(v any) any {
	for {
		inner, ok := v.(dbus.Variant)
		if !ok {
			return v
		}
		v = inner.Value()
	}
}
