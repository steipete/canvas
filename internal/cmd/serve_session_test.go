package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/steipete/canvas/internal/rpc"
	"github.com/steipete/canvas/internal/state"
)

func TestServePreservesLiveSession(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)
	stopped := false
	socket, shutdown := startUnixRPCServer(t, "test-token", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StatusResponse{Running: true})
		})
		mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) { stopped = true })
	})
	defer shutdown()
	if err := state.Save(stateDir, state.Session{PID: 1, SocketPath: socket, Token: "test-token"}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(state.SessionPath(stateDir))
	if err != nil {
		t.Fatal(err)
	}
	cmd := newServeCmd(&rootFlags{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--dir", t.TempDir()})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("serve: %v", err)
	}
	after, err := os.ReadFile(state.SessionPath(stateDir))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || stopped {
		t.Fatal("live session disturbed")
	}
}
