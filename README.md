# canvas

Note: I integrated this idea into https://github.com/steipete/clawdis - for now that's the main point where I'll iterate on this.
Canvas still has a great use case for less integrated personal assistants, my use case for now is in Clawdis tho.

`canvas` is a small Go tool that gives an agent a “visual workspace”:

- Serves a directory over HTTP (defaults to a new temp dir).
- Launches a controlled Chromium tab (single-tab).
- Exposes simple CLI commands to navigate, run JavaScript, query/modify DOM, take screenshots, and reload.
- Auto-reloads the tab when files on disk change.
- When the served directory has no `index.html` yet, Canvas shows a built-in welcome page.

This is intentionally flexible: an agent can write HTML/CSS/JS to disk, view it in a real browser, validate it via DOM/JS queries, and capture screenshots.

## Install / Build

Requires Go (this repo currently targets Go 1.25+).

Build:

```sh
go build ./cmd/canvas
```

Version stamping (optional):

```sh
go build -ldflags "-X github.com/steipete/canvas/internal/cmd.version=$(git rev-parse --short HEAD)" ./cmd/canvas
```

## Quickstart

Start a background session (headed by default):

```sh
canvas start
```

Restart the session (useful after updating Canvas, or to relaunch Chrome):

```sh
canvas start --restart
```

If you prefer a short “build + start” shortcut (requires `pnpm`):

```sh
pnpm canvas
```

Show status (use `--json` for agent-friendly output):

```sh
canvas status --json
```

Get the DevTools websocket URL (useful for external CDP clients):

```sh
canvas devtools
```

Write files into the session directory (`dir` from `canvas status --json`), then navigate:

```sh
canvas goto /
canvas goto /yolo
```

DOM + JS:

```sh
# Backward-compatible shorthand (equivalent to `canvas dom query ...`)
canvas dom "#app" --mode outer_html
canvas dom "#title" --mode text

# Explicit form
canvas dom query "#app" --mode outer_html
canvas eval "document.title"
```

DOM interactions:

```sh
canvas dom all "li" --mode text
canvas dom attr "#btn" "aria-label"
canvas dom click "#btn"
canvas dom type "#search" "hello" --clear
canvas dom wait "#result" --state visible --timeout 10s
```

Screenshots:

```sh
canvas screenshot --out /tmp/canvas.png
canvas screenshot --selector "#app" --out /tmp/app.png
```

Stop the session:

```sh
canvas stop
```

## Routing model

The served directory is mapped directly to URL paths:

- `/` serves `<dir>/index.html` (or `index.htm`) if present
- `/yolo` serves `<dir>/yolo/index.html` (or `index.htm`) if present
- other paths serve files directly (e.g. `/assets/app.css` -> `<dir>/assets/app.css`)

Directory listings are not enabled.

If `/` doesn’t have an `index.html` yet, Canvas serves a built-in welcome page instead of a plain 404 (helpful for first run).

## Commands

- `canvas start`: daemonizes (writes session info under the state dir)
- `canvas serve`: foreground mode (useful for debugging)
- `canvas status`: shows whether a session is running
- `canvas stop` (alias: `close`): stops server + closes controlled browser
- `canvas focus`: brings the controlled browser window to the front (macOS; no-op in headless)
- `canvas devtools`: prints DevTools websocket URL (or just the port)
- `canvas goto`: navigate to a path (e.g. `/yolo`) or full URL
- `canvas eval`: evaluate JavaScript
- `canvas dom`: DOM utilities (`query`, `all`, `attr`, `click`, `type`, `wait`)
- `canvas screenshot`: capture a PNG screenshot (full page or selector)
- `canvas reload`: reload the page

## DevTools (remote debugging)

Canvas launches Chromium with remote debugging bound to `127.0.0.1` and a dedicated port (random by default; override with `--devtools-port` on `canvas start`/`canvas serve`).

- `canvas status --json` includes `devtools_port` and `devtools_ws_url`
- `canvas devtools` prints the websocket URL (preferred) or the port

## State / configuration

State is stored under the platform config dir:

- macOS: `~/Library/Application Support/canvas/`

You can override this with:

- `CANVAS_STATE_DIR=/path/to/state`

Debug logging for the browser controller:

- `CANVAS_DEBUG=1`

## Platform

Primary target is macOS (headed mode is the default).

By default, the headed browser is launched in app mode (chromeless) — disable with `--app=false`.

Canvas also applies a best-effort “stealth” configuration to reduce automation detection signals — disable with `--stealth=false`.

## Roadmap

- Attach to an existing Chrome/Chromium session (instead of always launching our own).
- Screenshot options: no viewport sizing, full-page vs viewport toggle, JPEG, clip rect, DPR control.
- File server features: no SPA fallback, no custom headers, no directory listing toggle, no 404 page.
- Session robustness: no stale-session cleanup, PID validation, or “restart” command; daemon log exists but no `canvas logs`.
