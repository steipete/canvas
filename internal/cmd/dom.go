package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newDomCmd(root *rootFlags) *cobra.Command {
	var mode string

	cmd := &cobra.Command{
		Use:   "dom <css-selector>",
		Short: "Query the DOM (outer HTML by default)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := mustClient()
			if err != nil {
				return err
			}
			if mode == "" {
				mode = "outer_html"
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			out, err := c.Dom(ctx, args[0], mode)
			cancel()
			if err != nil {
				return err
			}
			if root.jsonOutput {
				return printJSON(out)
			}
			fmt.Fprintln(os.Stdout, out.Value)
			return nil
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "outer_html", "Query mode: outer_html or text")
	return cmd
}
