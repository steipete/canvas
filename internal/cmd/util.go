package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/steipete/canvas/internal/rpc"
	"github.com/steipete/canvas/internal/state"
)

func loadSession() (state.Session, string, error) {
	stateDir, err := state.DefaultStateDir()
	if err != nil {
		return state.Session{}, "", err
	}
	s, err := state.Load(stateDir)
	return s, stateDir, err
}

func mustClient() (*rpc.Client, state.Session, string, error) {
	s, stateDir, err := loadSession()
	if err != nil {
		return nil, state.Session{}, stateDir, err
	}
	return rpc.NewUnixClient(s.SocketPath, s.Token), s, stateDir, nil
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(b))
	return nil
}

func awaitDaemonReady(stateDir string, timeout time.Duration) (state.Session, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s, err := state.Load(stateDir)
		if err == nil && s.SocketPath != "" {
			c := rpc.NewUnixClient(s.SocketPath, s.Token)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, stErr := c.Status(ctx)
			cancel()
			if stErr == nil {
				return s, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return state.Session{}, errors.New("timed out waiting for daemon")
}
