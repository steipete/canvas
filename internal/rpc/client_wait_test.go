package rpc

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestDomWaitSurvivesClientTimeoutShorterThanRequest(t *testing.T) {
	socketPath := shortSocketPath(t)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	const (
		clientCap   = 100 * time.Millisecond
		elementAt   = 400 * time.Millisecond
		requestedMS = 1000
	)

	h := NewHandler("token123")
	h.Mux.HandleFunc("/dom/wait", func(w http.ResponseWriter, r *http.Request) {
		timer := time.NewTimer(elementAt)
		defer timer.Stop()
		select {
		case <-r.Context().Done():
			http.Error(w, "wait canceled", http.StatusGatewayTimeout)
			return
		case <-timer.C:
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(DomWaitResponse{OK: true, State: "visible"})
	})
	srv := &http.Server{Handler: h}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	c := NewUnixClient(socketPath, "token123")
	if c.httpClient.Timeout != 30*time.Second {
		t.Fatalf("default client timeout = %s", c.httpClient.Timeout)
	}
	// Same field as the 30s cap, shortened so the test does not sleep 30s.
	c.httpClient.Timeout = clientCap

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(requestedMS)*time.Millisecond)
	defer cancel()
	out, err := c.DomWait(ctx, "#late", "visible", requestedMS)
	if err != nil {
		t.Fatalf("DomWait: %v", err)
	}
	if !out.OK {
		t.Fatalf("response = %+v", out)
	}
	if c.httpClient.Timeout < time.Duration(requestedMS)*time.Millisecond {
		t.Fatalf("client timeout %s is still below the requested wait", c.httpClient.Timeout)
	}
}
