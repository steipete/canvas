package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

func newScreenshotCmd(root *rootFlags) *cobra.Command {
	var (
		selector string
		outPath  string
	)

	cmd := &cobra.Command{
		Use:   "screenshot",
		Short: "Take a screenshot of the controlled tab",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := mustClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			out, err := c.Screenshot(ctx, selector)
			cancel()
			if err != nil {
				return err
			}

			b, err := base64.StdEncoding.DecodeString(out.Base64)
			if err != nil {
				return err
			}

			if outPath == "" {
				outPath = fmt.Sprintf("canvas-%d.png", time.Now().UnixNano())
			}
			outPath = filepath.Clean(outPath)
			if err := os.WriteFile(outPath, b, 0o644); err != nil {
				return err
			}

			if root.jsonOutput {
				return printJSON(map[string]any{"path": outPath, "bytes": len(b)})
			}
			fmt.Fprintln(os.Stdout, outPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&selector, "selector", "", "CSS selector to screenshot (default: full page)")
	cmd.Flags().StringVar(&outPath, "out", "", "Output file path (default: canvas-<ts>.png)")
	return cmd
}
