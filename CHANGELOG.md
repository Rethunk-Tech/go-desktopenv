# Changelog

Notable changes to `go-desktopenv`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-10-30

### Added

- `locale.Language` reads the user's UI language as a tag such as `de-DE` (the Windows preferred UI language, or `LC_ALL`, `LC_MESSAGES` and `LANG` elsewhere), and `locale.FromPOSIX` converts a POSIX locale string to a tag.

## [0.3.1] - 2026-10-30

### Added

- `schemehandler.Config.Run` replaces the program runner (xdg-mime, update-desktop-database, the previous owner's program) so a consumer's tests can record the calls; nil keeps the real one.
- `nativemessaging.Host.Run` does the same for the programs the host manifests' Flatpak path runs.

## [0.3.0] - 2026-10-30

### Added

- `schemehandler` registers a program for URL schemes through a desktop entry and xdg-mime defaults on Linux (honouring AppImage integrators and the Flatpak host) or HKCU classes on Windows, hands each scheme back to its previous handler, forwards a URL to it, and reports who owns a scheme.
- `nativemessaging` writes, removes and checks native-messaging host manifests for Chromium-family browsers and Firefox on Linux and Windows, and reads and writes the length-prefixed stdio frames with a size limit.

## [0.2.1] - 2026-10-30

### Added

- `lnk.Target` reads the program or file a Windows `.lnk` launches through `WScript.Shell`.

## [0.2.0] - 2026-10-30

### Added

- `hostenv.Environ` gives a child process the environment without the variables an AppImage's AppRun injected.
- `fonts.List` lists the installed font families from fontconfig or the Windows registry.
- `sysaccent.Read` reads the desktop's accent colour from the DWM or the Settings portal.
- `lnk` writes a Windows `.lnk` and reads its arguments through `WScript.Shell`, and lists a folder's shortcuts.

## [0.1.0] - 2026-10-29

### Added

- `nowindow.Set` starts a console child of a GUI process without a console window on Windows; a no-op elsewhere.
- `desktopentry.Value`, `Exec` and `SplitExec` escape a Desktop Entry string value, build an `Exec=` value from an argv, and split one back, checked against `desktop-file-validate`.
- `flatpak.Inside`, `HostArgv`, `Command`, `CommandContext` and `HostLookPath` detect a Flatpak sandbox and run commands on the host through `flatpak-spawn`, ending the host child on cancel.
- `userdirs` resolves the config and data folders for a home (XDG on Linux, `%APPDATA%` and `%LOCALAPPDATA%` on Windows), the host user's folders from inside a Flatpak, and Flatpak app roots; an environment value counts only when absolute and only for the real home.
- `portal` calls xdg-desktop-portal: `SetAutostart`, `InstallLauncher`, `UninstallLauncher`, `LauncherExists`, `RetrieveSecret` and `ReadSetting`, each bounded by a context and answering `ErrUnsupported` off Linux.

[Unreleased]: https://github.com/Rethunk-Tech/go-desktopenv/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/Rethunk-Tech/go-desktopenv/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/Rethunk-Tech/go-desktopenv/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/Rethunk-Tech/go-desktopenv/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/Rethunk-Tech/go-desktopenv/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/Rethunk-Tech/go-desktopenv/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Rethunk-Tech/go-desktopenv/releases/tag/v0.1.0
