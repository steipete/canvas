package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/steipete/canvas/internal/rpc"
	"github.com/steipete/canvas/internal/state"
)

func newStatusCmd(root *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current canvas session status",
		RunE: func(cmd *cobra.Command, args []string) error {
			stateDir, err := state.DefaultStateDir()
			if err != nil {
				return err
			}
			s, err := state.Load(stateDir)
			if err != nil {
				if root.jsonOutput {
					return printJSON(rpc.StatusResponse{Running: false})
				}
				fmt.Fprintln(os.Stdout, "not running")
				return nil
			}

			c := rpc.NewUnixClient(s.SocketPath, s.Token)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			st, stErr := c.Status(ctx)
			cancel()
			if stErr != nil {
				if root.jsonOutput {
					return printJSON(rpc.StatusResponse{Running: false, Error: stErr.Error()})
				}
				fmt.Fprintln(os.Stdout, "not running")
				return nil
			}

			if root.jsonOutput {
				return printJSON(st)
			}
			fmt.Fprintf(os.Stdout, "running: http://%s:%d/\n", st.HTTPAddr, st.HTTPPort)
			fmt.Fprintf(os.Stdout, "dir: %s\n", st.Dir)
			fmt.Fprintf(os.Stdout, "url: %s\n", st.CurrentURL)
			return nil
		},
	}
	return cmd
}
