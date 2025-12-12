package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newReloadCmd(root *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reload",
		Short: "Reload the controlled tab",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := mustClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			out, err := c.Reload(ctx)
			cancel()
			if err != nil {
				return err
			}
			if root.jsonOutput {
				return printJSON(out)
			}
			fmt.Fprintln(os.Stdout, "ok")
			return nil
		},
	}
	return cmd
}
