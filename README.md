# canvas

`canvas` is a small Go tool that gives an agent a “visual workspace”:

- Serves a directory over HTTP (defaults to a new temp dir).
- Launches a controlled Chromium tab (single-tab).
- Exposes simple CLI commands to navigate, run JavaScript, query DOM, take screenshots, and reload.
- Auto-reloads the tab when files on disk change.

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

Show status (use `--json` for agent-friendly output):

```sh
canvas status --json
```

Write files into the session directory (`dir` from `canvas status --json`), then navigate:

```sh
canvas goto /
canvas goto /yolo
```

DOM + JS:

```sh
canvas dom "#app" --mode outer_html
canvas dom "#title" --mode text
canvas eval "document.title"
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

## Commands

- `canvas start`: daemonizes (writes session info under the state dir)
- `canvas serve`: foreground mode (useful for debugging)
- `canvas status`: shows whether a session is running
- `canvas stop` (alias: `close`): stops server + closes controlled browser
- `canvas focus`: brings the controlled browser window to the front (macOS; no-op in headless)
- `canvas goto`: navigate to a path (e.g. `/yolo`) or full URL
- `canvas eval`: evaluate JavaScript
- `canvas dom`: query DOM by CSS selector
- `canvas screenshot`: capture a PNG screenshot (full page or selector)
- `canvas reload`: reload the page

## State / configuration

State is stored under the platform config dir:

- macOS: `~/Library/Application Support/canvas/`

You can override this with:

- `CANVAS_STATE_DIR=/path/to/state`

Debug logging for the browser controller:

- `CANVAS_DEBUG=1`

## Platform

Primary target is macOS (headed mode is the default). Other platforms can build, but you’ll likely want to pass `--browser-bin` explicitly.

## Roadmap

- DevTools port / websocket URL exposure: right now we control Chrome via `chromedp` but we don’t surface a “DevTools port” you can attach to (nor do we support attaching to an existing Chrome).
