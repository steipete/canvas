package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/steipete/canvas/internal/osx"
)

func newFocusCmd(root *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "focus",
		Short: "Bring the controlled browser window to the front (macOS)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, sess, _, err := mustClient()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			st, stErr := c.Status(ctx)
			cancel()
			if stErr != nil || !st.Running {
				if root.jsonOutput {
					return printJSON(map[string]any{"ok": false, "running": false})
				}
				fmt.Fprintln(os.Stdout, "not running")
				return nil
			}
			if st.Headless {
				if root.jsonOutput {
					return printJSON(map[string]any{"ok": true, "headless": true})
				}
				fmt.Fprintln(os.Stdout, "ok (headless)")
				return nil
			}

			pid := sess.BrowserPID
			if pid == 0 {
				pid = st.BrowserPID
			}
			if err := osx.FocusPID(pid); err != nil {
				return err
			}

			if root.jsonOutput {
				return printJSON(map[string]any{"ok": true})
			}
			fmt.Fprintln(os.Stdout, "ok")
			return nil
		},
	}
	return cmd
}
