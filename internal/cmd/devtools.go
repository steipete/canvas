package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newDevToolsCmd(root *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devtools",
		Short: "Print DevTools debugging port / websocket URL for the controlled browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := mustClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			st, err := c.Status(ctx)
			cancel()
			if err != nil {
				return err
			}

			if root.jsonOutput {
				return printJSON(map[string]any{
					"devtools_port":   st.DevToolsPort,
					"devtools_ws_url": st.DevToolsWSURL,
				})
			}
			if st.DevToolsWSURL != "" {
				fmt.Fprintln(os.Stdout, st.DevToolsWSURL)
				return nil
			}
			fmt.Fprintln(os.Stdout, st.DevToolsPort)
			return nil
		},
	}
	return cmd
}
