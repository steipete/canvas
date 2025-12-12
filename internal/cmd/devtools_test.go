package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/steipete/canvas/internal/rpc"
	"github.com/steipete/canvas/internal/state"
)

func TestDevToolsCommand_JSON(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	socketPath := shortSocketPath(t)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	h := rpc.NewHandler("token123")
	h.Mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rpc.StatusResponse{
			Running:       true,
			DevToolsPort:  5555,
			DevToolsWSURL: "ws://127.0.0.1:5555/devtools/browser/abc",
		})
	})

	srv := &http.Server{Handler: h}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	if err := state.Save(stateDir, state.Session{
		PID:        1,
		SocketPath: socketPath,
		Token:      "token123",
	}); err != nil {
		t.Fatal(err)
	}

	flags := &rootFlags{jsonOutput: true}
	cmd := newDevToolsCmd(flags)
	cmd.SetArgs([]string{})

	var buf bytes.Buffer
	restore, err := captureStdout(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Execute(); err != nil {
		_ = restore()
		t.Fatal(err)
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v output=%q", err, buf.String())
	}
	if out["devtools_ws_url"] != "ws://127.0.0.1:5555/devtools/browser/abc" {
		t.Fatalf("unexpected devtools_ws_url: %#v", out["devtools_ws_url"])
	}
}
