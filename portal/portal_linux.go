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
	notifyIf   = "org.freedesktop.portal.Notification"
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
		dbus.WithMatchSender(busName),
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
		if sig.Path != handle {
			continue
		}
		code, results, err := decodeResponse(sig.Body)
		if err != nil {
			return nil, err
		}
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

// decodeResponse reads Request.Response's (u a{sv}). A missing field or a wrong type is the response, not a stray
// signal: the match already names this handle.
func decodeResponse(body []any) (uint32, map[string]dbus.Variant, error) {
	if len(body) < 2 {
		return 0, nil, errors.New("the portal response was truncated")
	}
	code, ok := body[0].(uint32)
	if !ok {
		return 0, nil, errors.New("the portal response code is not a uint32")
	}
	results, ok := body[1].(map[string]dbus.Variant)
	if !ok {
		return 0, nil, errors.New("the portal response results are not a map")
	}
	return code, results, nil
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
	icon := bytesIcon(l.IconPNG)
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

// bytesIcon is a serialized GBytesIcon: ("bytes", <PNG data>).
func bytesIcon(png []byte) dbus.Variant {
	return dbus.MakeVariant(struct {
		Kind string
		Data dbus.Variant
	}{"bytes", dbus.MakeVariant(png)})
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
	if dbusErr, ok := errors.AsType[dbus.Error](err); ok && strings.HasSuffix(dbusErr.Name, "_g_2dfile_2derror_2dquark.Code4") {
		return false, nil
	}
	return false, err
}

const (
	// maxSecret bounds the bytes read from the secret pipe; the portal's secret is 64 bytes.
	maxSecret = 4096
	// secretTimeout bounds the pipe read of a caller whose context has no deadline.
	secretTimeout = 30 * time.Second
)

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
	// The portal writes and closes the pipe; one that keeps it open or sends more than a secret is a failure.
	if dl, ok := ctx.Deadline(); ok {
		_ = r.SetReadDeadline(dl)
	} else {
		_ = r.SetReadDeadline(time.Now().Add(secretTimeout))
	}
	secret, err := io.ReadAll(io.LimitReader(r, maxSecret+1))
	if err != nil {
		return nil, err
	}
	if len(secret) > maxSecret {
		return nil, errors.New("the secret portal returned more than a secret")
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
	if dbusErr, ok := errors.AsType[dbus.Error](err); ok && dbusErr.Name == notFoundIf {
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

// AddNotification shows n through the Notification portal, which needs no talk-name on the notification service.
// The portal answers no Request: it returns once it has accepted the notification.
func AddNotification(ctx context.Context, n Notification) error {
	if n.ID == "" || n.Title == "" {
		return errors.New("portal: notification needs an id and a title")
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	options := map[string]dbus.Variant{"title": dbus.MakeVariant(n.Title)}
	if n.Body != "" {
		options["body"] = dbus.MakeVariant(n.Body)
	}
	if n.Priority != "" {
		options["priority"] = dbus.MakeVariant(string(n.Priority))
	}
	if len(n.IconPNG) > 0 {
		options["icon"] = bytesIcon(n.IconPNG)
	}
	if n.DefaultAction != "" {
		options["default-action"] = dbus.MakeVariant(n.DefaultAction)
		if n.DefaultActionTarget != "" {
			options["default-action-target"] = dbus.MakeVariant(n.DefaultActionTarget)
		}
	}
	if len(n.Buttons) > 0 {
		options["buttons"] = dbus.MakeVariant(buttonDicts(n.Buttons))
	}
	return call(ctx, conn, notifyIf+".AddNotification", n.ID, options).Err
}

// buttonDicts encodes buttons as the portal's aa{sv}; its spec makes label and action mandatory for a button
// without a purpose (https://flatpak.github.io/xdg-desktop-portal/docs/doc-org.freedesktop.portal.Notification.html).
func buttonDicts(buttons []NotificationButton) []map[string]dbus.Variant {
	out := make([]map[string]dbus.Variant, 0, len(buttons))
	for _, b := range buttons {
		d := map[string]dbus.Variant{"label": dbus.MakeVariant(b.Label), "action": dbus.MakeVariant(b.Action)}
		if b.Target != "" {
			d["target"] = dbus.MakeVariant(b.Target)
		}
		out = append(out, d)
	}
	return out
}

// OnNotificationAction calls fn, on its own goroutine, for each ActionInvoked signal the portal sends: a click on a
// notification's non-exported default action or button. Actions named "app.…" are exported and the portal activates
// them through D-Bus activation instead, so they never arrive here. It holds one session-bus connection until ctx
// ends or the returned stop is called; stop is idempotent and does not wait for a running fn.
func OnNotificationAction(ctx context.Context, fn func(NotificationAction)) (func(), error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	// Matching the sender keeps another bus client from forging a click.
	if err := conn.AddMatchSignal(
		dbus.WithMatchSender(busName),
		dbus.WithMatchObjectPath(objectPath),
		dbus.WithMatchInterface(notifyIf),
		dbus.WithMatchMember("ActionInvoked"),
	); err != nil {
		_ = conn.Close()
		return nil, err
	}
	signals := make(chan *dbus.Signal, 8)
	conn.Signal(signals)
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer func() { _ = conn.Close() }()
		for {
			select {
			case <-ctx.Done():
				return
			case sig, ok := <-signals:
				if !ok {
					return
				}
				if a, ok := decodeAction(sig.Body); ok {
					fn(a)
				}
			}
		}
	}()
	return cancel, nil
}

// decodeAction reads ActionInvoked's (id, action, parameter). The parameter array holds the button's target only when
// the notification set one, then the platform-data vardict, then a purpose response; a leading map therefore means no
// target.
func decodeAction(body []any) (NotificationAction, bool) {
	if len(body) < 3 {
		return NotificationAction{}, false
	}
	id, ok1 := body[0].(string)
	action, ok2 := body[1].(string)
	params, ok3 := body[2].([]dbus.Variant)
	if !ok1 || !ok2 || !ok3 {
		return NotificationAction{}, false
	}
	a := NotificationAction{ID: id, Action: action}
	if len(params) > 0 {
		if _, isMap := params[0].Value().(map[string]dbus.Variant); !isMap {
			a.Target = fmt.Sprint(unwrap(params[0]))
			params = params[1:]
		}
	}
	if len(params) > 0 {
		if pd, ok := params[0].Value().(map[string]dbus.Variant); ok {
			a.ActivationToken, _ = pd["activation-token"].Value().(string)
		}
	}
	if len(params) > 1 {
		a.Response, _ = params[1].Value().(string)
	}
	return a, true
}

// RemoveNotification withdraws a notification AddNotification showed; the portal ignores an unknown id.
func RemoveNotification(ctx context.Context, id string) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	return call(ctx, conn, notifyIf+".RemoveNotification", id).Err
}
