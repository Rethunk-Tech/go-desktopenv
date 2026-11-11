# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's private vulnerability reporting on this repository: <https://github.com/Rethunk-Tech/go-desktopenv/security/advisories/new>, or the **Security** tab, then **Report a vulnerability**. The report stays private between you and us until a fix ships.

If private reporting is unavailable, email [oss@rethunk.tech](mailto:oss@rethunk.tech) with **[SECURITY]** in the subject.

Please do not open a public issue, pull request or discussion for a suspected vulnerability.

A useful report has the affected repository and version (or commit), steps to reproduce, what an attacker gets and under which conditions, and whether anyone else knows.

## What counts as a vulnerability

A vulnerability is a flaw that lets someone cross a boundary they should not: run code, read or change data they have no right to, escape a sandbox or path restriction, or leak a credential. A bug is anything else that misbehaves, such as a crash on your own input, wrong output or a slow path; open an ordinary issue for those. If you are unsure which one you have, report it privately and we will say.

## Coordinated disclosure

We handle the report inside the private advisory, so you see each status change and can comment on the fix. The advisory goes public when the fix ships, or 90 days after your report, whichever comes first. If a fix is close, either of us can ask for more time in the thread. You are credited in the advisory unless you would rather not be.

## Response timelines

- We acknowledge your report within 3 business days.
- We give you an assessment (accepted, needs more information or declined, with reasons) within 10 business days.
- We aim to ship a fix within 90 days of the report.

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
- `nativemessaging.Read` reads frames a browser extension sends: a length over the caller's limit is refused before
  any body is allocated, a short stream is an error, and no input panics (`FuzzRead`).
- `schemehandler.Forward` hands a URL from the command line to the previous handler as one argv element: it is never
  split, never passed through a shell, and shell metacharacters in it stay literal. Inside a Flatpak it reaches
  the host's `gio` as a positional argument to a fixed script.
- `schemehandler` and `nativemessaging` reach host files inside a Flatpak through fixed `sh -c` scripts with every
  path and body as a positional argument. `DropDefault` removes only lines naming the handler's own entry.
- `schemehandler` reads other programs' desktop entries and registry values to name and forward to them: malformed
  ones give an error or an empty name, never a panic.

## What is out of scope

- The session bus and its portal service are trusted; a process able to impersonate the portal can already read the
  user's session.
- Names passed to `HostArgv`, `Command` and `HostLookPath` run on the host with the app's host access: the caller
  decides what may be run.
- `RetrieveSecret` returns key material; storing it safely is the caller's job.
- `SplitExec` reading `%%u` as `%u` is documented behaviour.
- The handler a user registered before this program is trusted: `Forward` runs it, as the desktop would.
- Which extensions may connect is the manifest's allow-list, which the caller supplies; the browser enforces it.
- The size of a URL on the command line is bounded by the OS; capping what the caller does with it is the caller's job.

## Safe harbour

We will not pursue legal action against good-faith research that avoids privacy violations, data destruction and disruption of services, stops at a proof of concept, and keeps the details private until the advisory is published. Rethunk.Tech, LLC maintains this repository.
