package rpc

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestClientStatus_DevToolsFields(t *testing.T) {
	socketPath := t.TempDir() + "/rpc.sock"
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	h := NewHandler("token123")
	h.Mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(StatusResponse{
			Running:       true,
			DevToolsPort:  12345,
			DevToolsWSURL: "ws://127.0.0.1:12345/devtools/browser/abc",
		})
	})

	srv := &http.Server{Handler: h}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	c := NewUnixClient(socketPath, "token123")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	st, err := c.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st.DevToolsPort != 12345 {
		t.Fatalf("devtools port = %d", st.DevToolsPort)
	}
	if st.DevToolsWSURL == "" {
		t.Fatalf("missing devtools ws url")
	}
}
