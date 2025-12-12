package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/steipete/canvas/internal/daemon"
	"github.com/steipete/canvas/internal/state"
)

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
