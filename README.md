<h1 align="center">go-desktopenv</h1>

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.3+-00ADD8?logo=go&logoColor=white)](go.mod)

</div>

---

What a packaged Go desktop app needs to reach its desktop: xdg-desktop-portal calls, Flatpak detection and host commands, per-user folders, console-less Windows children and Desktop Entry Exec values. Module `github.com/Rethunk-Tech/go-desktopenv`, five packages: `portal`, `flatpak`, `userdirs`, `desktopentry` and `nowindow`. Import only the ones you need; `flatpak`, `userdirs`, `desktopentry` and `nowindow` pull in no third-party code.

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
