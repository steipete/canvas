package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/steipete/canvas/internal/state"
)

func newStopCmd(root *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stop",
		Aliases: []string{"close"},
		Short:   "Stop canvas (server + controlled browser)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, stateDir, err := mustClient()
			if err != nil {
				// Not running.
				if root.jsonOutput {
					return printJSON(map[string]any{"running": false})
				}
				fmt.Fprintln(os.Stdout, "not running")
				return nil
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, _ = c.Stop(ctx)
			cancel()
			_ = state.Remove(stateDir)

			if root.jsonOutput {
				return printJSON(map[string]any{"ok": true})
			}
			fmt.Fprintln(os.Stdout, "stopped")
			return nil
		},
	}
	return cmd
}
