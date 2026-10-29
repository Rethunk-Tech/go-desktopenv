# Changelog

Notable changes to `go-desktopenv`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `nowindow.Set` starts a console child of a GUI process without a console window on Windows; a no-op elsewhere.
- `desktopentry.Value`, `Exec` and `SplitExec` escape a Desktop Entry string value, build an `Exec=` value from an argv, and split one back, checked against `desktop-file-validate`.
- `flatpak.Inside`, `HostArgv`, `Command`, `CommandContext` and `HostLookPath` detect a Flatpak sandbox and run commands on the host through `flatpak-spawn`, ending the host child on cancel.

[Unreleased]: https://github.com/Rethunk-Tech/go-desktopenv/commits/main
