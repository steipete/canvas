package cmd

import (
	"github.com/spf13/cobra"

	"github.com/steipete/canvas/internal/daemon"
)

func newDaemonCmd() *cobra.Command {
	var cfg daemon.Config

	cmd := &cobra.Command{
		Use:    "daemon",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Watch = true
			return daemon.Run(cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.StateDir, "state-dir", "", "State directory")
	cmd.Flags().StringVar(&cfg.ServeDir, "dir", "", "Directory to serve")
	cmd.Flags().IntVar(&cfg.HTTPPort, "port", 0, "HTTP port")
	cmd.Flags().IntVar(&cfg.DevToolsPort, "devtools-port", 0, "DevTools remote debugging port (0 picks a random free port)")
	cmd.Flags().BoolVar(&cfg.Headless, "headless", false, "Run browser headless")
	cmd.Flags().StringVar(&cfg.BrowserBin, "browser-bin", "", "Chromium/Chrome binary path (optional)")
	cmd.Flags().BoolVar(&cfg.TempDir, "temp-dir", false, "Remove served directory on shutdown")

	_ = cmd.MarkFlagRequired("state-dir")
	_ = cmd.MarkFlagRequired("dir")
	return cmd
}
