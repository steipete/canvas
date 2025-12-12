package browser

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestDevToolsWebSocketURL(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"webSocketDebuggerUrl":"ws://127.0.0.1:1234/devtools/browser/abc"}`))
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	port := ln.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := devToolsWebSocketURL(ctx, port)
	if err != nil {
		t.Fatal(err)
	}
	if got != "ws://127.0.0.1:1234/devtools/browser/abc" {
		t.Fatalf("ws url mismatch: %q", got)
	}
}
