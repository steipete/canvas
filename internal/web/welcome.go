package web

import (
	"fmt"
	"html"
	"strings"
)

type WelcomeData struct {
	ServeDir      string
	HTTPURL       string
	DevToolsPort  int
	DevToolsWSURL string
	AutoReload    bool
	AppMode       bool
}

type WelcomeProvider func() WelcomeData

func RenderWelcomeHTML(d WelcomeData) []byte {
	title := "Canvas"
	subtitle := "Write an index.html to get started."
	devTools := ""
	if d.DevToolsWSURL != "" {
		devTools = d.DevToolsWSURL
	} else if d.DevToolsPort != 0 {
		devTools = fmt.Sprintf("127.0.0.1:%d", d.DevToolsPort)
	}

	lines := []string{
		`<!doctype html>`,
		`<html lang="en">`,
		`<head>`,
		`<meta charset="utf-8" />`,
		`<meta name="viewport" content="width=device-width, initial-scale=1" />`,
		`<title>` + html.EscapeString(title) + `</title>`,
		`<style>`,
		`:root { color-scheme: light dark; }`,
		`body { font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial; margin: 0; padding: 32px; }`,
		`.card { max-width: 920px; margin: 0 auto; padding: 24px; border: 1px solid rgba(127,127,127,.25); border-radius: 14px; }`,
		`h1 { margin: 0 0 6px 0; font-size: 28px; }`,
		`p { margin: 0 0 14px 0; line-height: 1.5; opacity: .9 }`,
		`code, pre { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace; }`,
		`pre { background: rgba(127,127,127,.12); padding: 12px 14px; border-radius: 10px; overflow: auto; }`,
		`.grid { display: grid; grid-template-columns: 1fr; gap: 12px; }`,
		`.kv { display: grid; grid-template-columns: 140px 1fr; gap: 10px; }`,
		`.k { opacity: .75 }`,
		`.pill { display: inline-block; padding: 2px 8px; border: 1px solid rgba(127,127,127,.25); border-radius: 999px; font-size: 12px; opacity: .85 }`,
		`</style>`,
		`</head>`,
		`<body>`,
		`<div class="card">`,
		`<div class="pill">canvas</div>`,
		`<h1>` + html.EscapeString(title) + `</h1>`,
		`<p>` + html.EscapeString(subtitle) + `</p>`,
		`<div class="grid">`,
		`<div class="kv"><div class="k">Serving</div><div><code>` + html.EscapeString(d.ServeDir) + `</code></div></div>`,
	}

	if d.HTTPURL != "" {
		lines = append(lines, `<div class="kv"><div class="k">URL</div><div><code>`+html.EscapeString(d.HTTPURL)+`</code></div></div>`)
	}
	if devTools != "" {
		lines = append(lines, `<div class="kv"><div class="k">DevTools</div><div><code>`+html.EscapeString(devTools)+`</code></div></div>`)
	}
	if d.AutoReload {
		lines = append(lines, `<div class="kv"><div class="k">Reload</div><div>Auto-reload is on (write files to refresh)</div></div>`)
	}
	if d.AppMode {
		lines = append(lines, `<div class="kv"><div class="k">Window</div><div>App mode (chromeless)</div></div>`)
	}

	lines = append(lines,
		`</div>`,
		`<h2 style="margin: 18px 0 10px 0; font-size: 16px;">Create your first page</h2>`,
		`<pre><code>`+html.EscapeString(`cat > index.html <<'HTML'
<!doctype html>
<html>
  <body style="font-family: system-ui; padding: 24px">
    <h1>Hello Canvas</h1>
  </body>
</html>
HTML`)+`</code></pre>`,
		`<p>Then navigate with <code>canvas goto /</code>. DOM helpers: <code>canvas dom all "h1" --mode text</code>.</p>`,
		`</div>`,
		`</body>`,
		`</html>`,
	)

	return []byte(strings.Join(lines, "\n"))
}
