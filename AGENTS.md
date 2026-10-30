# AGENTS.md

Eleven packages a packaged Go desktop app uses to reach its desktop: `portal` (xdg-desktop-portal over D-Bus),
`flatpak` (sandbox detection and host commands), `userdirs` (per-user folders), `desktopentry` (Desktop Entry
values), `nowindow` (console-less Windows children), `hostenv` (a child's environment outside an AppImage's),
`fonts` (installed families), `sysaccent` (the desktop's accent colour), `lnk` (Windows shortcut files),
`schemehandler` (URL scheme registration) and `nativemessaging` (browser native-messaging hosts). It imports no consumer package, no GUI toolkit and no
branding; names, ids and wording are caller arguments. `portal` alone imports godbus; `nowindow`, `fonts`, `sysaccent`, `schemehandler` and `nativemessaging` import
`golang.org/x/sys` and `lnk` imports go-ole, on Windows only, so a consumer of another package gains none of them.
`schemehandler` and `nativemessaging` import go-safefs and share `internal/hostfile` for the host's files.
`sysaccent` reads through `portal` on Linux, and `fonts` runs `fc-list` with `hostenv`'s environment.

## Gate

`gate` is the command. Each override in `.gate.toml` is there because a runner
cannot resolve that check from PATH. The `windows-latest` test job is the only place
Windows-only behaviour runs: `nowindow`'s and `lnk`'s tests, the case-folded home test in `userdirs`, and the
registry tests of `schemehandler` and `nativemessaging`, which write under a fresh `HKCU\Software\DesktopEnvTest…` key. `portal`'s D-Bus tests
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
- `flatpak.Inside` is `FLATPAK_ID`, not `/.flatpak-info`: Flatpak sets it for every app, and tests can set it.
- `HostLookPath` asks the host's `sh` (`command -v`), not `which`, which minimal hosts lack, and rejects a
  non-absolute answer because `command -v` prints a builtin by its bare name.
- `desktopentry.Exec` always quotes `argv[0]`, so a reader can find the program path by its quoted form whatever
  characters it holds.
- `SplitExec` undoes `%%` before the caller sees field codes; a literal `%%u` is indistinguishable from `%u`.
  Accepted: no known entry carries one.
- `schemehandler` runs `/usr/bin/xdg-mime` by absolute path outside a Flatpak, so its Linux tests swap the
  Handler's own `run` field for a recorder; that field is per value, not a package seam.
- `schemehandler`'s desktop entry lists the schemes only while they are claimed, so a desktop that picks a default
  from advertised types does not pick the app after a Restore. `WriteEntry` compares the whole file, so the entry's
  text must stay byte-stable across releases or every caller rewrites it on start.
- An AppImage integrator's entry (GearLever, AppImageLauncher, appimaged) for the same program is the handler, and
  the app's own entry is removed with its defaults moved there; one whose Exec sets `DESKTOPINTEGRATION` wins.
- The previous Windows handler is saved as `command\nicon\nname` in `Owner.ID`, so Restore can put back all three.
- Inside a Flatpak, mimeapps.list and browser manifests are the host's, reached through `flatpak-spawn --host sh`
  scripts with paths as positional arguments (`internal/hostfile`), and a Flatpak counts as `Packaged`.
- `nativemessaging` frames use native byte order, as the Chrome and Firefox docs specify; every platform Go builds
  a browser for is little-endian. `Read` consumes the whole frame before decoding, so a bad body never misaligns the
  next frame.

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
| `hostenv/hostenv.go` | `Environ` and the AppImage scrub |
| `fonts/fonts.go` | `ParseFcList`, `ParseWindowsNames`; `fonts_linux.go` and `fonts_windows.go` hold `List` |
| `sysaccent/sysaccent.go` | `FromDWM`, `FromPortal`; `Read` per platform |
| `lnk/lnk.go` | `Shortcut`, `Files`; `lnk_windows.go` holds `Write`, `Arguments` and `Target`, `lnk_other.go` the `ErrUnsupported` stubs |
| `internal/hostfile/hostfile.go` | The host's files and commands, directly or through `flatpak-spawn --host` |
| `schemehandler/schemehandler.go` | `Config`, `Entry`, `Owner`, `ErrUnsupported` |
| `schemehandler/forward.go` | The forward argv for a desktop Exec or a Windows open command, the saved Windows owner |
| `schemehandler/schemehandler_linux.go`, `_windows.go`, `_other.go` | `Handler` and its calls per platform |
| `nativemessaging/nativemessaging.go` | `Host`, `Manifest`, `State`, `Status`, `Read`, `Write` |
| `nativemessaging/nativemessaging_linux.go`, `_windows.go`, `_other.go` | `Install`, `Uninstall`, `Status` per platform |
