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
| `userdirs` functions | An environment value only when home is the process's own home and the value is absolute; otherwise the default under home. `""` for an empty home. Never reads `runtime.GOOS`. |
| `userdirs.HostConfigHome`, `HostDataHome` | Inside a Flatpak, read `HOST_XDG_*`, not the app's own `XDG_*`. |
| `desktopentry.Exec` | `argv[0]` is always quoted; every `%` is doubled; field codes are the caller's to append. |
| `desktopentry.SplitExec` | Undoes string escapes, quoting and `%%`, so a literal `%%u` and the field code `%u` both read back as `%u`. Other field codes stay for the caller. |
| `nowindow.Set` | Adds `CREATE_NO_WINDOW` and keeps the existing `SysProcAttr`; a no-op off Windows. |

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
