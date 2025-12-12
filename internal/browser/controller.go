package browser

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"github.com/chromedp/chromedp"
)

type Controller struct {
	mu            sync.Mutex
	allocCtx      context.Context
	tabCtx        context.Context
	cancelAll     context.CancelFunc
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

	allocOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	allocOpts = append(allocOpts,
		chromedp.ExecPath(bin),
		chromedp.Flag("headless", opts.Headless),
		chromedp.Flag("remote-debugging-address", "127.0.0.1"),
		chromedp.Flag("remote-debugging-port", fmt.Sprintf("%d", devToolsPort)),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
	)
	if opts.UserDataDir != "" {
		allocOpts = append(allocOpts, chromedp.UserDataDir(opts.UserDataDir))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, allocOpts...)
	ctxOpts := []chromedp.ContextOption{}
	if os.Getenv("CANVAS_DEBUG") != "" {
		ctxOpts = append(ctxOpts, chromedp.WithDebugf(log.Printf), chromedp.WithLogf(log.Printf), chromedp.WithErrorf(log.Printf))
	}
	tabCtx, tabCancel := chromedp.NewContext(allocCtx, ctxOpts...)

	allCancel := func() {
		tabCancel()
		allocCancel()
	}

	c := &Controller{
		allocCtx:     allocCtx,
		tabCtx:       tabCtx,
		cancelAll:    allCancel,
		browserBin:   bin,
		headless:     opts.Headless,
		devToolsPort: devToolsPort,
	}

	// Ensure the browser is actually up.
	if err := chromedp.Run(c.tabCtx, chromedp.Navigate("about:blank")); err != nil {
		c.cancelAll()
		return nil, fmt.Errorf("chromedp startup failed: %w", err)
	}

	if chromedp.FromContext(c.tabCtx) == nil || chromedp.FromContext(c.tabCtx).Browser == nil || chromedp.FromContext(c.tabCtx).Browser.Process() == nil {
		c.cancelAll()
		return nil, errors.New("chromedp startup failed: missing browser process")
	}
	c.browserPID = chromedp.FromContext(c.tabCtx).Browser.Process().Pid

	wsURL, _ := DevToolsWebSocketURL(devToolsPort)
	c.devToolsWSURL = wsURL

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
