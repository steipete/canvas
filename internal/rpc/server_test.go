package rpc

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_Auth(t *testing.T) {
	h := NewHandler("token123")
	h.Mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	// No auth => 401.
	resp, err := http.Get(srv.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	// With auth => 200.
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer token123")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%q", resp2.StatusCode, string(b))
	}
}
