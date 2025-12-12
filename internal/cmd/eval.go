package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func newEvalCmd(root *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval <js-expression>",
		Short: "Evaluate JavaScript in the controlled tab",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := mustClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			out, err := c.Eval(ctx, args[0])
			cancel()
			if err != nil {
				return err
			}
			if root.jsonOutput {
				return printJSON(out)
			}

			switch v := out.Value.(type) {
			case string:
				fmt.Fprintln(os.Stdout, v)
			default:
				b, _ := json.MarshalIndent(out.Value, "", "  ")
				fmt.Fprintln(os.Stdout, string(b))
			}
			return nil
		},
	}
	return cmd
}
