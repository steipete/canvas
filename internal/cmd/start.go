package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/steipete/canvas/internal/rpc"
	"github.com/steipete/canvas/internal/state"
)

func newStartCmd(root *rootFlags) *cobra.Command {
	var (
		dir          string
		port         int
		devToolsPort int
		headless     bool
		app          bool
		windowSize   string
		browserBin   string
		stealth      bool
		restart      bool
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start canvas in the background (daemon)",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir, err := state.DefaultStateDir()
			if err != nil {
				return err
			}

			// If already running, just report status (unless restarting).
			if s, err := state.Load(stateDir); err == nil && s.SocketPath != "" {
				c := rpc.NewUnixClient(s.SocketPath, s.Token)
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				st, stErr := c.Status(ctx)
				cancel()
				if stErr == nil && st.Running {
					if restart {
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						_, _ = c.Stop(ctx)
						cancel()
						_ = state.Remove(stateDir)
					} else {
						if root.jsonOutput {
							return printJSON(st)
						}
						fmt.Fprintf(os.Stdout, "running: http://%s:%d/\n", st.HTTPAddr, st.HTTPPort)
						fmt.Fprintf(os.Stdout, "dir: %s\n", st.Dir)
						if st.DevToolsWSURL != "" {
							fmt.Fprintf(os.Stdout, "devtools: %s\n", st.DevToolsWSURL)
						} else if st.DevToolsPort != 0 {
							fmt.Fprintf(os.Stdout, "devtools-port: %d\n", st.DevToolsPort)
						}
						return nil
					}
				} else if restart || stErr == nil || statusEndpointDead(stErr) {
					// Stale session; remove and proceed.
					_ = state.Remove(stateDir)
				} else {
					// Not a dead socket. Leave the running daemon in place.
					return fmt.Errorf("the existing daemon did not answer in time: %w", stErr)
				}
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

			if err := os.MkdirAll(stateDir, 0o700); err != nil {
				return err
			}

			logPath := filepath.Join(stateDir, "daemon.log")
			logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			defer logFile.Close()

			args2 := []string{
				"daemon",
				"--state-dir", stateDir,
				"--dir", dir,
				"--port", fmt.Sprintf("%d", port),
				"--devtools-port", fmt.Sprintf("%d", devToolsPort),
				"--app", fmt.Sprintf("%t", app),
				"--stealth", fmt.Sprintf("%t", stealth),
				"--window-size", windowSize,
			}
			if headless {
				args2 = append(args2, "--headless")
			}
			if browserBin != "" {
				args2 = append(args2, "--browser-bin", browserBin)
			}
			if tempDir {
				args2 = append(args2, "--temp-dir")
			}

			if err := spawnDaemon(os.Args[0], args2, logFile); err != nil {
				return err
			}

			sess, err := awaitDaemonReady(stateDir, 15*time.Second)
			if err != nil {
				return fmt.Errorf("%w (see %s)", err, logPath)
			}

			c := rpc.NewUnixClient(sess.SocketPath, sess.Token)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			st, _ := c.Status(ctx)
			cancel()

			if root.jsonOutput {
				return printJSON(st)
			}
			fmt.Fprintf(os.Stdout, "running: http://%s:%d/\n", st.HTTPAddr, st.HTTPPort)
			fmt.Fprintf(os.Stdout, "dir: %s\n", st.Dir)
			if st.DevToolsWSURL != "" {
				fmt.Fprintf(os.Stdout, "devtools: %s\n", st.DevToolsWSURL)
			} else if st.DevToolsPort != 0 {
				fmt.Fprintf(os.Stdout, "devtools-port: %d\n", st.DevToolsPort)
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
	cmd.Flags().BoolVar(&restart, "restart", false, "Restart if already running")

	return cmd
}

// statusEndpointDead reports a failed dial to the saved socket: the file
// is missing, the connection is refused, or connect fails the same way.
// A timeout, or an error after the socket accepts, is not proof it is dead.
func statusEndpointDead(err error) bool {
	if err == nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return false
	}
	if errors.Is(err, syscall.ENOENT) || errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "dial"
}

var spawnDaemon = func(bin string, args []string, logFile *os.File) error {
	proc := exec.Command(bin, args...)
	proc.Stdout = logFile
	proc.Stderr = logFile
	proc.Stdin = nil
	proc.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return proc.Start()
}
