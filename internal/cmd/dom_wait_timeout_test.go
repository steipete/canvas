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

func TestDomWaitCommand_TimeoutLongerThanClientCap(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	prev := rpc.DefaultHTTPTimeout
	rpc.DefaultHTTPTimeout = 100 * time.Millisecond
	t.Cleanup(func() { rpc.DefaultHTTPTimeout = prev })

	socketPath := shortSocketPath(t)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	h := rpc.NewHandler("token123")
	h.Mux.HandleFunc("/dom/wait", func(w http.ResponseWriter, r *http.Request) {
		timer := time.NewTimer(400 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-r.Context().Done():
			http.Error(w, "wait canceled", http.StatusGatewayTimeout)
			return
		case <-timer.C:
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rpc.DomWaitResponse{OK: true, State: "visible"})
	})
	srv := &http.Server{Handler: h}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	if err := state.Save(stateDir, state.Session{PID: 1, SocketPath: socketPath, Token: "token123"}); err != nil {
		t.Fatal(err)
	}

	cmd := newDomCmd(&rootFlags{})
	cmd.SetArgs([]string{"wait", "#late", "--timeout", "1s"})

	var buf bytes.Buffer
	restore, err := captureStdout(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Execute(); err != nil {
		_ = restore()
		t.Fatalf("dom wait: %v; output %q", err, buf.String())
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "ok\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}
