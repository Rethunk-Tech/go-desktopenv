# HUMANS.md

Run and use `go-desktopenv`. Internals: [AGENTS.md](AGENTS.md).

## Quick start

```bash
go get github.com/Rethunk-Tech/go-desktopenv
```

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Rethunk-Tech/go-desktopenv/desktopentry"
	"github.com/Rethunk-Tech/go-desktopenv/flatpak"
	"github.com/Rethunk-Tech/go-desktopenv/portal"
	"github.com/Rethunk-Tech/go-desktopenv/userdirs"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Ask for autostart; the dialog waits on the user until ctx ends.
	err := portal.SetAutostart(ctx, portal.Autostart{
		Enable:  true,
		Command: []string{"myapp", "--background"},
		Reason:  "Start myapp when you sign in",
	})
	fmt.Println(err) // portal.ErrCancelled, portal.ErrNotAllowed or portal.ErrUnsupported are values to test

	// Run a command on the host from inside a Flatpak; outside one it runs directly.
	if out, err := flatpak.Command(ctx, "uname", "-r").Output(); err == nil {
		fmt.Printf("%s", out)
	}

	// Another program's config folder, as the host sees it.
	fmt.Println(userdirs.HostConfigHome("/home/me"))

	// An Exec= value for a .desktop file you write yourself.
	fmt.Println("Exec=" + desktopentry.Exec("/opt/my app/myapp", "--play=a b") + " %u")
}
```

## Contracts

| Entry point | Contract |
| --- | --- |
| `portal` calls | Open the session bus, make one request, close it. The context bounds the whole call; one still waiting on its dialog when the context ends gets `Request.Close` and returns `ctx.Err()`. No timeout is imposed. |
| `portal.ErrCancelled`, `ErrNotAllowed`, `ErrNotFound`, `ErrUnsupported` | Compare with `errors.Is`. Declared on every platform; off Linux every call returns `ErrUnsupported`. |
| `portal.SetAutostart` | `ErrNotAllowed` when enabling was not granted. An empty `Reason` sends none. |
| `portal.InstallLauncher` | Builds the entry itself through `desktopentry`; an empty `Exec` is an error. The name the dialog returns wins over `Name`. |
| `portal.UninstallLauncher`, `LauncherExists` | A launcher the portal does not know is `false` or no error, never a failure. |
| `portal.RetrieveSecret` | The same bytes on every call for one app id; an empty secret is an error. Works only inside a Flatpak. |
| `portal.ReadSetting` | The value with every enclosing variant removed; `ErrNotFound` when the portal has none. |
| `flatpak.Inside` | True when `FLATPAK_ID` is set. |
| `flatpak.Command` | `flatpak-spawn --host` inside a Flatpak, the command itself outside. `Dir` and `Env` set on the inside Cmd do not reach the host program: use `HostArgv`. |
| `flatpak.CommandContext` | For `flatpak-spawn`, cancel sends SIGTERM (which it forwards to the host child) and `WaitDelay` is 5 s. |
| `flatpak.HostLookPath` | An absolute path, or `exec.ErrNotFound`; a shell builtin's bare name is not a path. Always spawns `flatpak-spawn`. |
| `userdirs` functions | An environment value only when home is the process's own home and the value is absolute; otherwise the default under home. `""` for an empty home. Never reads `runtime.GOOS`. A test that fakes a home sets `USERPROFILE` beside `HOME`, since `os.UserHomeDir` reads it on Windows. |
| `userdirs.HostConfigHome`, `HostDataHome` | Inside a Flatpak, read `HOST_XDG_*`, not the app's own `XDG_*`. |
| `desktopentry.Exec` | `argv[0]` is always quoted; every `%` is doubled; field codes are the caller's to append. |
| `desktopentry.SplitExec` | Undoes string escapes, quoting and `%%`, so a literal `%%u` and the field code `%u` both read back as `%u`. Other field codes stay for the caller. |
| `nowindow.Set` | Adds `CREATE_NO_WINDOW` and keeps the existing `SysProcAttr`; a no-op off Windows. |
| `hostenv.Environ` | `os.Environ`, except inside an AppImage (`APPIMAGE` and `APPDIR` set) on Linux: path-list entries under `$APPDIR` are dropped, a variable they empty is dropped, and so are `APPDIR`, `APPIMAGE` and `OWD`. |
| `fonts.List` | Sorted, case-insensitively unique family names: `fc-list` (10 s limit) on Linux, the machine and user fonts registry keys on Windows, `nil` elsewhere. `ParseFcList` and `ParseWindowsNames` take the raw text on any platform. |
| `sysaccent.Read` | `"#RRGGBB"` and true, or false when the desktop sets none: the DWM `AccentColor` on Windows, the portal's `accent-color` (5 s limit) on Linux, false elsewhere. `FromDWM` and `FromPortal` convert the raw values on any platform. |
| `lnk.Write`, `Arguments`, `Target` | Windows only (`ErrUnsupported` elsewhere). `Write` replaces the file; `Target` reads the launched path back; `Target` and `Arguments` are always set, so an empty `Arguments` clears the old ones, and an empty `WorkingDir`, `Icon` or `Description` is left unset. |
| `lnk.Files` | The `.lnk` files directly in a folder; a missing folder holds none. Any platform. |
| `schemehandler.New` | A `Handler` for one `Config`; on Linux it reads `HOME`, `XDG_DATA_HOME` and `XDG_DATA_DIRS` once. Off Linux and Windows every call is `ErrUnsupported`. |
| `Handler.Register` | Linux: writes the user entry listing every `Config.Schemes` (or yields to an AppImage integrator's entry for `Exe`), then sets the given schemes' defaults. Windows: `"Exe" "%1"` under `HKCU\Software\Classes\<scheme>`. |
| `Handler.Owner` | Facts only: the owner's `ID` (to save for Restore), display `Name`, and whether it is `Exe`. An empty `ID` is no owner. Wording and repair are the caller's. |
| `Handler.Restore`, `Release` | Hand each scheme back to its saved `ID`, or drop the default when there is none. Linux: `Restore` rewrites the entry without the schemes; `Release` keeps them listed. Windows: a scheme another program owns is left alone. |
| `Handler.Forward` | Runs the saved owner on the URL as one argument, never through a shell: the entry's Exec with `%u`/`%U` replaced (gio on the host inside a Flatpak), or the Windows command with `%1` replaced, started without a console and not waited for. |
| `Handler.SetDefault`, `DropDefault`, `WriteEntry`, `EntryProgram`, `RemoveEntry` | Linux only. `DropDefault` edits mimeapps.list through a symlink; `WriteEntry` writes only when the text differs; `EntryProgram` is `fs.ErrNotExist` without an entry. A `Packaged` Config (or a Flatpak) writes no user entry. |
| `nativemessaging.Host.Install` | Linux: a manifest in each installed browser's folder (Chrome, Chromium, Brave, Edge, Vivaldi, Firefox), through a wrapper inside a Flatpak. Windows: two files in `ManifestDir` and an HKCU key per browser, installed or not. |
| `Host.Uninstall`, `Status` | `Uninstall` removes what `Install` wrote. `Status` lists installed browsers only, each `OK`, `Missing`, `Stale` (names another program) or `Unreadable`. |
| `nativemessaging.Read`, `Write` | One message: a 4-byte native-order length, then JSON. `Read` refuses a length over its limit before reading the body and gives `io.EOF` only between messages. |

`portal` tests run under a private session bus: they need `dbus-run-session`, and `desktopentry` tests use
`desktop-file-validate` when present. Both are required (the test fails) when `CI` is set. `TestLivePortal` calls
the real Secret portal and runs only inside a Flatpak with `DESKTOPENV_LIVE_PORTAL=1`.

## Requirements

Go 1.27.3 or newer, plus `lefthook` and `gitleaks` on PATH if you want the git hooks.

```bash
lefthook install
```

That is per clone; git does not commit hooks. Pre-commit scans staged changes for secrets, and
pre-push runs `gate`.

## Verify

```bash
gate
```
