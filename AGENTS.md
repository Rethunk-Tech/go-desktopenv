# AGENTS.md

Five packages a packaged Go desktop app uses to reach its desktop: `portal` (xdg-desktop-portal over D-Bus),
`flatpak` (sandbox detection and host commands), `userdirs` (per-user folders), `desktopentry` (Desktop Entry
values) and `nowindow` (console-less Windows children). It imports no consumer package, no GUI toolkit and no
branding; names, ids and wording are caller arguments. `portal` alone imports godbus, and `nowindow` alone imports
`golang.org/x/sys`, so a consumer of another package gains neither requirement.

## Gate

`gate` is the command. Each override in `.gate.toml` is there because a runner
cannot resolve that check from PATH. The `windows-latest` test job is the only place
Windows-only behaviour runs: `nowindow`'s tests and the case-folded home test in `userdirs`. `portal`'s D-Bus tests
and `flatpak`'s fake-`flatpak-spawn` tests run only on Linux (build tags `linux`, `unix`). `crosscompile` vets for
Windows and for darwin, which is what builds `portal_other.go` and `nowindow_other.go`.

## Shape

- No package-level mutable test seam. In a shared library a stub in one consumer's test rewires every other
  importer; tests drive `FLATPAK_ID`, `HOME` and `PATH` with `t.Setenv` and use real processes.
- `portal` has no timeout of its own: the caller's context is the only bound, since a library cannot pick one for
  every dialog. Every direct method call goes through `CallWithContext` (`call`), and `request` is the single path
  for calls answered through a Request object.
- `request` subscribes to the Response before calling, because the portal may answer before the call returns.
- Each portal call opens its own connection, so the handle token needs only to be fresh, not coordinated.
- `portal/errors.go` carries no build tag, so Windows and darwin builds declare the same sentinels.
- `userdirs` decides by `home`, never by `runtime.GOOS`: the caller knows which layout it is resolving. The
  environment describes only the real home, hence the real-home check and `""` for an empty home.
- `flatpak.Inside` is `FLATPAK_ID`, not `/.flatpak-info`: it is what tests can set and what released apps used.
- `HostLookPath` asks the host's `sh` (`command -v`), not `which`, which minimal hosts lack, and rejects a
  non-absolute answer because `command -v` prints a builtin by its bare name.
- `desktopentry.Exec` always quotes `argv[0]`: readers in released apps match the `Exec="` prefix.
- `SplitExec` undoes `%%` before the caller sees field codes; a literal `%%u` is indistinguishable from `%u`.
  Accepted: no known entry carries one.

## Files

| File | Holds |
| --- | --- |
| `portal/doc.go` | Package documentation |
| `portal/errors.go` | The sentinel errors and the `Autostart` and `Launcher` types, on every platform |
| `portal/portal_linux.go` | The D-Bus calls, the Request wait, the launcher entry builder |
| `portal/portal_other.go` | The same signatures returning `ErrUnsupported` |
| `flatpak/flatpak.go` | `Inside`, `HostArgv`, `Command`, `CommandContext`, `HostLookPath` |
| `userdirs/userdirs.go` | The folder functions and the one environment rule |
| `userdirs/foldcase_windows.go`, `foldcase_other.go` | Home comparison, case-folded on Windows |
| `desktopentry/desktopentry.go` | `Value`, `Exec`, `SplitExec` |
| `nowindow/nowindow_windows.go`, `nowindow_other.go` | `Set` |
