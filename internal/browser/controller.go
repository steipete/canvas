package browser

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"time"

	cdpbrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

type Controller struct {
	mu            sync.Mutex
	actionGate    chan struct{}
	runCDP        func(context.Context, ...chromedp.Action) error
	tabCtx        context.Context
	cancelAll     context.CancelFunc
	browserCmd    *exec.Cmd
	browserBin    string
	headless      bool
	browserPID    int
	devToolsPort  int
	devToolsWSURL string
}

type Options struct {
	BrowserBin   string
	Headless     bool
	UserDataDir  string
	DevToolsPort int
	StartURL     string
	AppMode      bool
	WindowSize   string
	Stealth      bool
}

func New(ctx context.Context, opts Options) (*Controller, error) {
	bin := opts.BrowserBin
	if bin == "" {
		var err error
		bin, err = FindChromiumBinary()
		if err != nil {
			return nil, err
		}
	}

	devToolsPort := opts.DevToolsPort
	if devToolsPort == 0 {
		p, err := pickFreeLocalPort()
		if err != nil {
			return nil, err
		}
		devToolsPort = p
	}

	if opts.WindowSize == "" {
		opts.WindowSize = "1280,720"
	}

	launched, err := launch(ctx, LaunchOptions{
		BrowserBin:   bin,
		Headless:     opts.Headless,
		UserDataDir:  opts.UserDataDir,
		DevToolsPort: devToolsPort,
		StartURL:     opts.StartURL,
		AppMode:      opts.AppMode,
		WindowSize:   opts.WindowSize,
	})
	if err != nil {
		return nil, err
	}

	tabCtx, cancel, err := newRemoteTabContext(ctx, launched.DevToolsWS, launched.TargetID)
	if err != nil {
		_ = terminateProcess(launched.Cmd, 2*time.Second)
		return nil, fmt.Errorf("chromedp attach failed: %w", err)
	}

	c := &Controller{
		tabCtx:        tabCtx,
		cancelAll:     cancel,
		browserCmd:    launched.Cmd,
		browserBin:    bin,
		headless:      opts.Headless,
		browserPID:    launched.Cmd.Process.Pid,
		devToolsPort:  launched.DevToolsPort,
		devToolsWSURL: launched.DevToolsWS,
	}

	if opts.Stealth {
		_ = c.applyStealth()
	}

	return c, nil
}

func (c *Controller) BrowserBinary() string { return c.browserBin }
func (c *Controller) Headless() bool        { return c.headless }
func (c *Controller) BrowserPID() int       { return c.browserPID }
func (c *Controller) DevToolsPort() int     { return c.devToolsPort }
func (c *Controller) DevToolsWSURL() string { return c.devToolsWSURL }

func pickFreeLocalPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func (c *Controller) Close() error {
	c.mu.Lock()
	if c.cancelAll == nil {
		c.mu.Unlock()
		return nil
	}
	tabCtx := c.tabCtx
	cancelAll := c.cancelAll
	cmd := c.browserCmd
	c.cancelAll = nil
	c.mu.Unlock()

	// The browser close is bounded. Canceling the tab context afterwards
	// unblocks a CDP call that is still waiting, without making that call
	// hold the controller lock.
	closeCtx, cancel := context.WithTimeout(tabCtx, 2*time.Second)
	closeErr := c.executeCDP(closeCtx, cdpbrowser.Close())
	cancel()
	cancelAll()
	if closeErr == nil {
		_ = waitProcess(cmd, 5*time.Second)
	} else {
		_ = terminateProcess(cmd, 2*time.Second)
	}
	return nil
}

func (c *Controller) run(ctx context.Context, timeout time.Duration, actions ...chromedp.Action) error {
	c.mu.Lock()
	tabCtx := c.tabCtx
	closed := c.cancelAll == nil
	// Serialize page actions without holding the shutdown lock.
	if c.actionGate == nil {
		c.actionGate = make(chan struct{}, 1)
	}
	gate := c.actionGate
	c.mu.Unlock()
	if tabCtx == nil || closed {
		return errors.New("browser closed")
	}

	var (
		runCtx context.Context
		cancel context.CancelFunc
	)
	if timeout > 0 {
		runCtx, cancel = context.WithTimeout(tabCtx, timeout)
	} else {
		runCtx, cancel = context.WithCancel(tabCtx)
	}
	defer cancel()
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
		stop := context.AfterFunc(ctx, cancel)
		defer stop()
	}
	select {
	case gate <- struct{}{}:
		defer func() { <-gate }()
	case <-runCtx.Done():
		return runCtx.Err()
	}
	if err := runCtx.Err(); err != nil {
		return err
	}
	return c.executeCDP(runCtx, actions...)
}

func (c *Controller) executeCDP(ctx context.Context, actions ...chromedp.Action) error {
	if c.runCDP != nil {
		return c.runCDP(ctx, actions...)
	}
	return chromedp.Run(ctx, actions...)
}

func (c *Controller) Alive(ctx context.Context) bool {
	var title string
	return c.run(ctx, 0, chromedp.Title(&title)) == nil
}

func (c *Controller) Navigate(ctx context.Context, url string) (string, string, error) {
	if url == "" {
		return "", "", errors.New("missing url")
	}
	var title string
	var loc string
	err := c.run(ctx, 0,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Location(&loc),
		chromedp.Title(&title),
	)
	return loc, title, err
}

func (c *Controller) Reload(ctx context.Context) error {
	return c.run(ctx, 0,
		chromedp.Reload(),
		chromedp.WaitReady("body", chromedp.ByQuery),
	)
}

func (c *Controller) Eval(ctx context.Context, expr string) (any, error) {
	var out any
	if err := c.run(ctx, 0, chromedp.Evaluate(expr, &out)); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Controller) OuterHTML(ctx context.Context, selector string) (string, error) {
	var out string
	if err := c.run(ctx, 0, chromedp.OuterHTML(selector, &out, chromedp.ByQuery)); err != nil {
		return "", err
	}
	return out, nil
}

func (c *Controller) Text(ctx context.Context, selector string) (string, error) {
	var out string
	if err := c.run(ctx, 0, chromedp.Text(selector, &out, chromedp.ByQuery)); err != nil {
		return "", err
	}
	return out, nil
}

func (c *Controller) Screenshot(ctx context.Context, selector string) ([]byte, error) {
	var buf []byte
	var action chromedp.Action
	if selector == "" {
		action = chromedp.CaptureScreenshot(&buf)
	} else {
		action = chromedp.Screenshot(selector, &buf, chromedp.NodeVisible, chromedp.ByQuery)
	}
	if err := c.run(ctx, 0, action); err != nil {
		return nil, err
	}
	return buf, nil
}

func (c *Controller) Location(ctx context.Context) (string, error) {
	var loc string
	if err := c.run(ctx, 0, chromedp.Location(&loc)); err != nil {
		return "", err
	}
	return loc, nil
}

func (c *Controller) Title(ctx context.Context) (string, error) {
	var title string
	if err := c.run(ctx, 0, chromedp.Title(&title)); err != nil {
		return "", err
	}
	return title, nil
}

func (c *Controller) applyStealth() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Best-effort only: different Chromium builds support different CDP features.
	_ = emulation.SetAutomationOverride(false).Do(c.tabCtx)
	_ = page.Enable().Do(c.tabCtx)
	_, err := page.AddScriptToEvaluateOnNewDocument(stealthScript).Do(c.tabCtx)
	return err
}

const stealthScript = `(function () {
  try {
    Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
  } catch (_) {}
  try {
    window.chrome = window.chrome || {};
  } catch (_) {}
})();`

func (c *Controller) QueryAll(ctx context.Context, selector, mode string) ([]string, error) {
	if selector == "" {
		return nil, errors.New("missing selector")
	}
	if mode == "" {
		mode = "outer_html"
	}

	exprSel := strconv.Quote(selector)
	var expr string
	switch mode {
	case "outer_html":
		expr = fmt.Sprintf(`Array.from(document.querySelectorAll(%s)).map(n => n.outerHTML)`, exprSel)
	case "text":
		expr = fmt.Sprintf(`Array.from(document.querySelectorAll(%s)).map(n => (n.textContent ?? ""))`, exprSel)
	default:
		return nil, errors.New("unknown mode")
	}

	var out []string
	if err := c.run(ctx, 15*time.Second, chromedp.Evaluate(expr, &out)); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Controller) Attr(ctx context.Context, selector, name string) (*string, error) {
	if selector == "" {
		return nil, errors.New("missing selector")
	}
	if name == "" {
		return nil, errors.New("missing name")
	}

	exprSel := strconv.Quote(selector)
	exprName := strconv.Quote(name)
	expr := fmt.Sprintf(`(() => { const el = document.querySelector(%s); if (!el) return {"__canvas":"not_found"}; return el.getAttribute(%s); })()`, exprSel, exprName)

	var out any
	if err := c.run(ctx, 15*time.Second, chromedp.Evaluate(expr, &out)); err != nil {
		return nil, err
	}
	switch v := out.(type) {
	case string:
		return &v, nil
	case nil:
		return nil, nil
	case map[string]any:
		return nil, errors.New("element not found")
	default:
		return nil, fmt.Errorf("unexpected attr result type %T", out)
	}
}

func (c *Controller) Click(ctx context.Context, selector string) error {
	if selector == "" {
		return errors.New("missing selector")
	}
	return c.run(ctx, 15*time.Second, chromedp.Click(selector, chromedp.ByQuery))
}

func (c *Controller) Type(ctx context.Context, selector, text string, clear bool) error {
	if selector == "" {
		return errors.New("missing selector")
	}
	actions := []chromedp.Action{
		chromedp.Focus(selector, chromedp.ByQuery),
	}
	if clear {
		// Assign directly: JSON v2 omits empty strings in CDP call arguments.
		expr := fmt.Sprintf(`(() => {
			const el = document.querySelector(%s);
			el.value = "";
			el.dispatchEvent(new Event('input', {bubbles: true}));
			el.dispatchEvent(new Event('change', {bubbles: true}));
		})()`, strconv.Quote(selector))
		actions = append(actions, chromedp.Evaluate(expr, nil))
	}
	actions = append(actions, chromedp.SendKeys(selector, text, chromedp.ByQuery))

	return c.run(ctx, 15*time.Second, actions...)
}

func (c *Controller) Wait(ctx context.Context, selector, state string, timeout time.Duration) error {
	if selector == "" {
		return errors.New("missing selector")
	}
	if state == "" {
		state = "visible"
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	var action chromedp.Action
	switch state {
	case "visible":
		action = chromedp.WaitVisible(selector, chromedp.ByQuery)
	case "hidden":
		action = chromedp.WaitNotVisible(selector, chromedp.ByQuery)
	case "ready":
		action = chromedp.WaitReady(selector, chromedp.ByQuery)
	case "present":
		action = chromedp.WaitReady(selector, chromedp.ByQuery)
	case "gone":
		action = chromedp.WaitNotPresent(selector, chromedp.ByQuery)
	default:
		return errors.New("unknown state")
	}

	return c.run(ctx, timeout, action)
}
