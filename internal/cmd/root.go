package cmd

import (
	"github.com/spf13/cobra"
)

var version = "dev"

type rootFlags struct {
	jsonOutput bool
}

func newRootCmd() *cobra.Command {
	var flags rootFlags

	cmd := &cobra.Command{
		Use:           "canvas",
		Short:         "Canvas: a controllable web + browser workspace for agents",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVar(&flags.jsonOutput, "json", false, "Output JSON when supported")
	cmd.Version = version
	cmd.SetVersionTemplate("{{.Version}}\n")

	cmd.AddCommand(
		newStartCmd(&flags),
		newServeCmd(&flags),
		newDaemonCmd(),
		newStatusCmd(&flags),
		newStopCmd(&flags),
		newFocusCmd(&flags),
		newDevToolsCmd(&flags),
		newGotoCmd(&flags),
		newEvalCmd(&flags),
		newReloadCmd(&flags),
		newDomCmd(&flags),
		newScreenshotCmd(&flags),
	)

	return cmd
}
