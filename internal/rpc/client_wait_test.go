package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDomWaitExtendsOnlyItsOwnTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(100 * time.Millisecond):
			_, _ = w.Write([]byte(`{"ok":true}`))
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	c := &Client{baseURL: server.URL, httpClient: &http.Client{Timeout: 20 * time.Millisecond}}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := c.DomWait(ctx, "#late", "visible", 500); done <- err }()
	if _, err := c.Status(ctx); err == nil {
		t.Fatal("ordinary RPC lost its timeout")
	}
	if err := <-done; err != nil {
		t.Fatalf("extended DOM wait: %v", err)
	}
	if c.httpClient.Timeout != 20*time.Millisecond {
		t.Fatal("shared timeout changed")
	}
}
