package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/steipete/canvas/internal/daemon"
	"github.com/steipete/canvas/internal/rpc"
	"github.com/steipete/canvas/internal/state"
)

// httpPortReleaseTimeout bounds the wait for /stop to drop its HTTP port.
// The handler answers before that listener closes.
var httpPortReleaseTimeout = 5 * time.Second

func newServeCmd(root *rootFlags) *cobra.Command {
	var (
		dir          string
		port         int
		devToolsPort int
		headless     bool
		app          bool
		windowSize   string
		browserBin   string
		stealth      bool
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run canvas in the foreground",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir, err := state.DefaultStateDir()
			if err != nil {
				return err
			}
			if err := stopDaemonBeforeServe(stateDir, port); err != nil {
				return err
			}

			tempDir := false
			if dir == "" {
				d, err := os.MkdirTemp("", "canvas-*")
				if err != nil {
					return err
				}
				dir = d
				tempDir = true
			} else {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return err
				}
			}

			if err := state.Remove(stateDir); err != nil {
				return err
			}

			cfg := daemon.Config{
				StateDir:     stateDir,
				ServeDir:     dir,
				HTTPPort:     port,
				DevToolsPort: devToolsPort,
				Headless:     headless,
				App:          app,
				WindowSize:   windowSize,
				BrowserBin:   browserBin,
				Stealth:      stealth,
				TempDir:      tempDir,
				Watch:        true,
			}

			if err := daemon.Run(cfg); err != nil && !errors.Is(err, os.ErrClosed) {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "Directory to serve (defaults to a temporary directory)")
	cmd.Flags().IntVar(&port, "port", 0, "HTTP port (0 picks a random free port)")
	cmd.Flags().IntVar(&devToolsPort, "devtools-port", 0, "DevTools remote debugging port (0 picks a random free port)")
	cmd.Flags().BoolVar(&headless, "headless", false, "Run browser headless")
	cmd.Flags().BoolVar(&app, "app", true, "Launch browser in app mode (chromeless window)")
	cmd.Flags().BoolVar(&stealth, "stealth", true, "Best-effort automation detection reduction")
	cmd.Flags().StringVar(&windowSize, "window-size", "1280,720", "Browser window size, e.g. 1280,720")
	cmd.Flags().StringVar(&browserBin, "browser-bin", "", "Chromium/Chrome binary path (optional)")
	return cmd
}

func stopDaemonBeforeServe(stateDir string, port int) error {
	sess, err := state.Load(stateDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	// A requested port other than the recorded listener is not ours.
	// Refuse before /stop so a busy port cannot take the live daemon down.
	if port > 0 && port != sess.HTTPPort {
		if err := httpPortFree(port); err != nil {
			return err
		}
	}

	stopped, err := stopRecordedDaemon(sess)
	if err != nil {
		return err
	}
	if !stopped {
		if port > 0 {
			return httpPortFree(port)
		}
		return nil
	}

	var ports []int
	if sess.HTTPPort > 0 {
		ports = append(ports, sess.HTTPPort)
	}
	if port > 0 && port != sess.HTTPPort {
		ports = append(ports, port)
	}
	if len(ports) == 0 {
		return nil
	}
	return waitUntilPortsFree(ports, httpPortReleaseTimeout)
}

func stopRecordedDaemon(sess state.Session) (bool, error) {
	if sess.SocketPath == "" {
		return false, nil
	}
	c := rpc.NewUnixClient(sess.SocketPath, sess.Token)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	st, stErr := c.Status(ctx)
	cancel()
	if stErr != nil || !st.Running {
		return false, nil
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()
	if _, err := c.Stop(stopCtx); err != nil {
		return false, fmt.Errorf("stop running canvas: %w", err)
	}
	return true, nil
}

func waitUntilPortsFree(ports []int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var last error
	for {
		last = nil
		for _, port := range ports {
			if err := httpPortFree(port); err != nil {
				last = err
				break
			}
		}
		if last == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return last
		}
		time.Sleep(15 * time.Millisecond)
	}
}

func httpPortFree(port int) error {
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)))
	if err != nil {
		return fmt.Errorf("http port %d is still in use: %w", port, err)
	}
	if err := ln.Close(); err != nil {
		return fmt.Errorf("release probed http port %d: %w", port, err)
	}
	return nil
}
