# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's **private vulnerability reporting** on this repository (Security → Report a
vulnerability). It stays private until a fix ships.

Please do not open a public issue for a suspected vulnerability first.

## What is in scope

- `desktopentry.SplitExec` reads any program's `.desktop` file: it must not panic or loop on malformed escapes or
  an unclosed quote.
- `desktopentry.Exec` and `Value` write values from caller data (paths, names): no input may end the key, open a
  second key, or change which program the entry runs. A line break never survives into the output.
- `portal.InstallLauncher` builds a Desktop Entry from caller fields through `desktopentry`: a `;` in a category
  must not add a category.
- `flatpak.HostArgv` and `Command` pass arguments to the host as separate argv elements, never through a shell;
  `HostLookPath` passes the name as a positional argument to `sh`, so it is never interpreted as shell code.
- `portal` replies (variants, the secret) are read from the user's own session bus: a reply of the wrong type is
  an error or a zero value, never a panic.
- `userdirs` must not honour a relative environment path or one describing a different home.

## What is out of scope

- The session bus and its portal service are trusted; a process able to impersonate the portal can already read the
  user's session.
- Names passed to `HostArgv`, `Command` and `HostLookPath` run on the host with the app's host access: the caller
  decides what may be run.
- `RetrieveSecret` returns key material; storing it safely is the caller's job.
- `SplitExec` reading `%%u` as `%u` is documented behaviour.
