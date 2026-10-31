<h1 align="center">go-desktopenv</h1>

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.3+-00ADD8?logo=go&logoColor=white)](go.mod)

</div>

---

What a packaged Go desktop app needs to reach its desktop: xdg-desktop-portal calls, Flatpak detection and host commands, per-user folders, console-less Windows children, Desktop Entry Exec values, an AppImage-free child environment, installed fonts, the system accent colour, the UI language, Windows shortcut files, URL scheme handling and browser native messaging. Module `github.com/Rethunk-Tech/go-desktopenv`, twelve packages: `portal`, `flatpak`, `userdirs`, `desktopentry`, `nowindow`, `hostenv`, `fonts`, `sysaccent`, `locale`, `lnk`, `schemehandler` and `nativemessaging`. Import only the ones you need; `portal` pulls in godbus, `lnk` go-ole on Windows, `nowindow`, `fonts`, `sysaccent`, `locale`, `schemehandler` and `nativemessaging` `golang.org/x/sys` on Windows, and `schemehandler` and `nativemessaging` go-safefs. The rest pull in no third-party code.

A Flatpak app needs `--talk-name=org.freedesktop.Flatpak` for `flatpak`; portals need nothing extra. A `portal` call waits on a user dialog until its context ends, so set a deadline. `RetrieveSecret` works only inside a Flatpak.

## Quick start

```bash
go get github.com/Rethunk-Tech/go-desktopenv
```

A worked example and the contracts: [HUMANS.md](HUMANS.md).

## Features

- `portal`: autostart, launchers, the per-app secret and desktop settings, each bounded by your context
- `flatpak`: detect a sandbox, run or build a command for the host, end the host child on cancel
- `userdirs`: config and data folders per home, honouring only absolute environment values for the real home
- `desktopentry`: escape, build and read Desktop Entry `Exec=` values
- `nowindow`: no console window for a console child of a Windows GUI process
- `hostenv`: a child process's environment without what an AppImage's AppRun injected
- `fonts`: the installed font families, from fontconfig or the Windows registry
- `sysaccent`: the desktop's accent colour as `#rrggbb`, from the DWM or the Settings portal
- `locale`: the user's UI language as a BCP-47-ish tag, from the Windows preferred UI languages or `LC_ALL`, `LC_MESSAGES` and `LANG`
- `lnk`: write a Windows `.lnk` with target, arguments, working folder, icon and description, read its arguments, list a folder's shortcuts
- `schemehandler`: register a program for URL schemes (desktop entry and xdg-mime defaults, or HKCU classes), yield to an AppImage integrator's entry, hand each scheme back to its previous handler, forward a URL to that handler, and report who owns a scheme
- `nativemessaging`: write, remove and check browser native-messaging host manifests for Chromium-family browsers and Firefox, and read and write the length-prefixed stdio frames

## Documentation

| Doc | Audience |
| --- | --- |
| [HUMANS.md](HUMANS.md) | Running and using the package |
| [AGENTS.md](AGENTS.md) | Internals, invariants, and why |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Process — commits, tests, docs |
| [SECURITY.md](SECURITY.md) | Vulnerability disclosure |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

## License

MIT — see [LICENSE](LICENSE). © 2026 Rethunk.Tech, LLC.
