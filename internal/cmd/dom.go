package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newDomCmd(root *rootFlags) *cobra.Command {
	var mode string

	cmd := &cobra.Command{
		Use:   "dom [command]",
		Short: "DOM utilities (query, queryAll, attrs, click, type, wait)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Backward-compatible: `canvas dom <selector>`
			if len(args) == 1 {
				return runDomQuery(root, args[0], mode)
			}
			return cmd.Help()
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "outer_html", "Query mode: outer_html or text")

	cmd.AddCommand(
		newDomQueryCmd(root, &mode),
		newDomAllCmd(root, &mode),
		newDomAttrCmd(root),
		newDomClickCmd(root),
		newDomTypeCmd(root),
		newDomWaitCmd(root),
	)

	return cmd
}

func newDomQueryCmd(root *rootFlags, mode *string) *cobra.Command {
	return &cobra.Command{
		Use:   "query <css-selector>",
		Short: "Query a single element (outer HTML by default)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDomQuery(root, args[0], *mode)
		},
	}
}

func runDomQuery(root *rootFlags, selector, mode string) error {
	c, _, _, err := mustClient()
	if err != nil {
		return err
	}
	if mode == "" {
		mode = "outer_html"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	out, err := c.Dom(ctx, selector, mode)
	cancel()
	if err != nil {
		return err
	}
	if root.jsonOutput {
		return printJSON(out)
	}
	fmt.Fprintln(os.Stdout, out.Value)
	return nil
}

func newDomAllCmd(root *rootFlags, mode *string) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "all <css-selector>",
		Short: "Query all matching elements",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := mustClient()
			if err != nil {
				return err
			}
			m := *mode
			if m == "" {
				m = "outer_html"
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			out, err := c.DomAll(ctx, args[0], m)
			cancel()
			if err != nil {
				return err
			}
			if limit > 0 && len(out.Values) > limit {
				out.Values = out.Values[:limit]
			}
			if root.jsonOutput {
				return printJSON(out)
			}
			if m == "text" {
				for _, v := range out.Values {
					fmt.Fprintln(os.Stdout, strings.TrimRight(v, "\n"))
				}
				return nil
			}
			// outer_html: separate entries for readability.
			for i, v := range out.Values {
				if i > 0 {
					fmt.Fprintln(os.Stdout, "---")
				}
				fmt.Fprintln(os.Stdout, v)
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Limit number of returned elements (0 = unlimited)")
	return cmd
}

func newDomAttrCmd(root *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "attr <css-selector> <name>",
		Short: "Get an attribute value from the first matching element",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := mustClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			out, err := c.DomAttr(ctx, args[0], args[1])
			cancel()
			if err != nil {
				return err
			}
			if root.jsonOutput {
				return printJSON(out)
			}
			if out.Value == nil {
				return nil
			}
			fmt.Fprintln(os.Stdout, *out.Value)
			return nil
		},
	}
}

func newDomClickCmd(root *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "click <css-selector>",
		Short: "Click the first matching element",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := mustClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			out, err := c.DomClick(ctx, args[0])
			cancel()
			if err != nil {
				return err
			}
			if root.jsonOutput {
				return printJSON(out)
			}
			if !out.OK {
				return errors.New("click failed")
			}
			fmt.Fprintln(os.Stdout, "ok")
			return nil
		},
	}
}

func newDomTypeCmd(root *rootFlags) *cobra.Command {
	var clear bool

	cmd := &cobra.Command{
		Use:   "type <css-selector> <text>",
		Short: "Type into the first matching element",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := mustClient()
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			out, err := c.DomType(ctx, args[0], args[1], clear)
			cancel()
			if err != nil {
				return err
			}
			if root.jsonOutput {
				return printJSON(out)
			}
			if !out.OK {
				return errors.New("type failed")
			}
			fmt.Fprintln(os.Stdout, "ok")
			return nil
		},
	}

	cmd.Flags().BoolVar(&clear, "clear", false, "Clear element value before typing")
	return cmd
}

func newDomWaitCmd(root *rootFlags) *cobra.Command {
	var (
		state   string
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "wait <css-selector>",
		Short: "Wait for a selector state (visible by default)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, _, err := mustClient()
			if err != nil {
				return err
			}
			if state == "" {
				state = "visible"
			}
			ms := 0
			if timeout > 0 {
				ms = int(timeout.Milliseconds())
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeoutOrDefault(timeout, 20*time.Second))
			out, err := c.DomWait(ctx, args[0], state, ms)
			cancel()
			if err != nil {
				return err
			}
			if root.jsonOutput {
				return printJSON(out)
			}
			if !out.OK {
				return fmt.Errorf("wait failed (%s)", state)
			}
			fmt.Fprintln(os.Stdout, "ok")
			return nil
		},
	}

	cmd.Flags().StringVar(&state, "state", "visible", "State: visible, hidden, ready, present, gone")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "Wait timeout")
	return cmd
}

func timeoutOrDefault(v, def time.Duration) time.Duration {
	if v > 0 {
		return v
	}
	return def
}
