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

	"github.com/chromedp/chromedp"
)

type Controller struct {
	mu            sync.Mutex
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
	defer c.mu.Unlock()
	if c.cancelAll == nil {
		return nil
	}
	c.cancelAll()
	_ = terminateProcess(c.browserCmd, 2*time.Second)
	c.cancelAll = nil
	return nil
}

func (c *Controller) Alive(ctx context.Context) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	var title string
	err := chromedp.Run(c.tabCtx, chromedp.Title(&title))
	return err == nil
}

func (c *Controller) Navigate(ctx context.Context, url string) (string, string, error) {
	if url == "" {
		return "", "", errors.New("missing url")
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	var title string
	var loc string
	err := chromedp.Run(c.tabCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Location(&loc),
		chromedp.Title(&title),
	)
	return loc, title, err
}

func (c *Controller) Reload(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return chromedp.Run(c.tabCtx,
		chromedp.Reload(),
		chromedp.WaitReady("body", chromedp.ByQuery),
	)
}

func (c *Controller) Eval(ctx context.Context, expr string) (any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out any
	if err := chromedp.Run(c.tabCtx, chromedp.Evaluate(expr, &out)); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Controller) OuterHTML(ctx context.Context, selector string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out string
	if err := chromedp.Run(c.tabCtx, chromedp.OuterHTML(selector, &out, chromedp.ByQuery)); err != nil {
		return "", err
	}
	return out, nil
}

func (c *Controller) Text(ctx context.Context, selector string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out string
	if err := chromedp.Run(c.tabCtx, chromedp.Text(selector, &out, chromedp.ByQuery)); err != nil {
		return "", err
	}
	return out, nil
}

func (c *Controller) Screenshot(ctx context.Context, selector string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var buf []byte
	var action chromedp.Action
	if selector == "" {
		action = chromedp.CaptureScreenshot(&buf)
	} else {
		action = chromedp.Screenshot(selector, &buf, chromedp.NodeVisible, chromedp.ByQuery)
	}
	if err := chromedp.Run(c.tabCtx, action); err != nil {
		return nil, err
	}
	return buf, nil
}

func (c *Controller) Location(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var loc string
	if err := chromedp.Run(c.tabCtx, chromedp.Location(&loc)); err != nil {
		return "", err
	}
	return loc, nil
}

func (c *Controller) Title(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var title string
	if err := chromedp.Run(c.tabCtx, chromedp.Title(&title)); err != nil {
		return "", err
	}
	return title, nil
}

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

	c.mu.Lock()
	defer c.mu.Unlock()
	runCtx, cancel := context.WithTimeout(c.tabCtx, 15*time.Second)
	defer cancel()

	var out []string
	if err := chromedp.Run(runCtx, chromedp.Evaluate(expr, &out)); err != nil {
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

	c.mu.Lock()
	defer c.mu.Unlock()
	runCtx, cancel := context.WithTimeout(c.tabCtx, 15*time.Second)
	defer cancel()

	var out any
	if err := chromedp.Run(runCtx, chromedp.Evaluate(expr, &out)); err != nil {
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
	c.mu.Lock()
	defer c.mu.Unlock()
	runCtx, cancel := context.WithTimeout(c.tabCtx, 15*time.Second)
	defer cancel()
	return chromedp.Run(runCtx, chromedp.Click(selector, chromedp.ByQuery))
}

func (c *Controller) Type(ctx context.Context, selector, text string, clear bool) error {
	if selector == "" {
		return errors.New("missing selector")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	runCtx, cancel := context.WithTimeout(c.tabCtx, 15*time.Second)
	defer cancel()

	actions := []chromedp.Action{
		chromedp.Focus(selector, chromedp.ByQuery),
	}
	if clear {
		actions = append(actions, chromedp.SetValue(selector, "", chromedp.ByQuery))
	}
	actions = append(actions, chromedp.SendKeys(selector, text, chromedp.ByQuery))

	return chromedp.Run(runCtx, actions...)
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

	c.mu.Lock()
	defer c.mu.Unlock()
	runCtx, cancel := context.WithTimeout(c.tabCtx, timeout)
	defer cancel()
	return chromedp.Run(runCtx, action)
}
