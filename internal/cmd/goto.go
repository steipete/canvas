package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newGotoCmd(root *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "goto <path-or-url>",
		Short: "Navigate the controlled tab to a path (e.g. /yolo) or full URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := mustClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			out, err := c.Goto(ctx, args[0])
			cancel()
			if err != nil {
				return err
			}
			if root.jsonOutput {
				return printJSON(out)
			}
			fmt.Fprintln(os.Stdout, out.URL)
			return nil
		},
	}
	return cmd
}
