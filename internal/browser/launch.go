package browser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

type LaunchOptions struct {
	BrowserBin   string
	Headless     bool
	UserDataDir  string
	DevToolsPort int

	// StartURL is used for app-mode (and for picking an initial target).
	StartURL string
	AppMode  bool

	WindowSize string // e.g. "1280,720"
}

type launchedBrowser struct {
	Cmd          *exec.Cmd
	DevToolsWS   string
	DevToolsPort int
	TargetID     target.ID
}

func launch(ctx context.Context, opts LaunchOptions) (launchedBrowser, error) {
	if opts.DevToolsPort == 0 {
		return launchedBrowser{}, errors.New("missing DevTools port")
	}
	if opts.BrowserBin == "" {
		return launchedBrowser{}, errors.New("missing browser bin")
	}

	args := []string{
		"--remote-debugging-address=127.0.0.1",
		fmt.Sprintf("--remote-debugging-port=%d", opts.DevToolsPort),
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-default-apps",
		"--disable-extensions",
		"--disable-popup-blocking",
		"--disable-infobars",
		"--disable-blink-features=AutomationControlled",
	}

	if opts.UserDataDir != "" {
		args = append(args, "--user-data-dir="+opts.UserDataDir)
	}

	if opts.WindowSize != "" {
		args = append(args, "--window-size="+opts.WindowSize)
	}

	if opts.Headless {
		// Prefer the modern headless mode when available.
		args = append(args, "--headless=new")
		args = append(args, "--disable-gpu")
	}

	startURL := opts.StartURL
	if startURL == "" {
		startURL = "about:blank"
	}
	if opts.AppMode && !opts.Headless && opts.StartURL != "" {
		args = append(args, "--app="+opts.StartURL)
	} else {
		args = append(args, startURL)
	}

	cmd := exec.CommandContext(ctx, opts.BrowserBin, args...)
	if os.Getenv("CANVAS_DEBUG") != "" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}

	if os.Getenv("CANVAS_DEBUG") != "" {
		log.Printf("launching browser: %s %s", opts.BrowserBin, strings.Join(args, " "))
	}

	if err := cmd.Start(); err != nil {
		return launchedBrowser{}, err
	}

	ws, err := DevToolsWebSocketURL(opts.DevToolsPort)
	if err != nil {
		_ = terminateProcess(cmd, 2*time.Second)
		return launchedBrowser{}, err
	}

	// Prefer attaching to the first "page" target (ideally the app/start URL).
	tgtID := target.ID("")
	targets, _ := DevToolsTargets(opts.DevToolsPort)
	if len(targets) > 0 {
		tgtID = pickTarget(targets, opts.StartURL)
	}

	return launchedBrowser{
		Cmd:          cmd,
		DevToolsWS:   ws,
		DevToolsPort: opts.DevToolsPort,
		TargetID:     tgtID,
	}, nil
}

func pickTarget(targets []DevToolsTarget, startURL string) target.ID {
	if startURL != "" {
		for _, t := range targets {
			if t.Type == "page" && t.URL == startURL && t.ID != "" {
				return target.ID(t.ID)
			}
		}
	}
	for _, t := range targets {
		if t.Type == "page" && t.ID != "" {
			return target.ID(t.ID)
		}
	}
	return ""
}

func newRemoteTabContext(parent context.Context, wsURL string, targetID target.ID) (context.Context, func(), error) {
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(parent, wsURL)

	ctxOpts := []chromedp.ContextOption{}
	if os.Getenv("CANVAS_DEBUG") != "" {
		ctxOpts = append(ctxOpts, chromedp.WithDebugf(log.Printf), chromedp.WithLogf(log.Printf), chromedp.WithErrorf(log.Printf))
	}
	if targetID != "" {
		ctxOpts = append(ctxOpts, chromedp.WithTargetID(targetID))
	}

	tabCtx, tabCancel := chromedp.NewContext(allocCtx, ctxOpts...)
	cancel := func() {
		tabCancel()
		allocCancel()
	}

	// Sanity check.
	if err := chromedp.Run(tabCtx); err != nil {
		cancel()
		return nil, nil, err
	}
	return tabCtx, cancel, nil
}

func terminateProcess(cmd *exec.Cmd, timeout time.Duration) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	_ = cmd.Process.Signal(syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		<-done
		return context.DeadlineExceeded
	}
}
