# TODO

Work that is decided but not built. `AGENTS.md` and `README.md` describe what
exists; this file is the only place that describes what does not.

## A Notification portal call when a consumer drops the Notifications talk-name

`portal` has no call for `org.freedesktop.portal.Notification`; a consumer that notifies today does it through
`org.freedesktop.Notifications` with that bus name in its finish-args. When one wants to drop the talk-name, add
`AddNotification` and `RemoveNotification` beside the other calls, on the same per-call connection and caller
context, with a fake in the D-Bus test and the live check behind `DESKTOPENV_LIVE_PORTAL`.
