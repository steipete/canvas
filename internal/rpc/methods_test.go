package rpc

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestClientMethods_RequestShapes(t *testing.T) {
	socketPath := shortSocketPath(t)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	h := NewHandler("token123")

	h.Mux.HandleFunc("/goto", func(w http.ResponseWriter, r *http.Request) {
		var req GotoRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = r.Body.Close()
		if req.URL != "/yolo" {
			t.Fatalf("goto url=%q", req.URL)
		}
		_ = json.NewEncoder(w).Encode(GotoResponse{URL: "http://x/yolo"})
	})

	h.Mux.HandleFunc("/eval", func(w http.ResponseWriter, r *http.Request) {
		var req EvalRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = r.Body.Close()
		if req.Expression == "" {
			t.Fatalf("missing expression")
		}
		_ = json.NewEncoder(w).Encode(EvalResponse{Value: map[string]any{"ok": true}})
	})

	h.Mux.HandleFunc("/dom/all", func(w http.ResponseWriter, r *http.Request) {
		var req DomAllRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = r.Body.Close()
		if req.Selector != "li" || req.Mode != "text" {
			t.Fatalf("dom/all=%#v", req)
		}
		_ = json.NewEncoder(w).Encode(DomAllResponse{Selector: req.Selector, Mode: req.Mode, Values: []string{"a"}})
	})

	h.Mux.HandleFunc("/dom/attr", func(w http.ResponseWriter, r *http.Request) {
		var req DomAttrRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = r.Body.Close()
		if req.Name != "data-x" {
			t.Fatalf("attr name=%q", req.Name)
		}
		v := "1"
		_ = json.NewEncoder(w).Encode(DomAttrResponse{Selector: req.Selector, Name: req.Name, Value: &v})
	})

	h.Mux.HandleFunc("/dom/click", func(w http.ResponseWriter, r *http.Request) {
		var req DomClickRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = r.Body.Close()
		if req.Selector != "#btn" {
			t.Fatalf("click selector=%q", req.Selector)
		}
		_ = json.NewEncoder(w).Encode(DomClickResponse{OK: true})
	})

	h.Mux.HandleFunc("/dom/type", func(w http.ResponseWriter, r *http.Request) {
		var req DomTypeRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = r.Body.Close()
		if req.Selector != "#in" || req.Text != "hello" || !req.Clear {
			t.Fatalf("type=%#v", req)
		}
		_ = json.NewEncoder(w).Encode(DomTypeResponse{OK: true})
	})

	h.Mux.HandleFunc("/dom/wait", func(w http.ResponseWriter, r *http.Request) {
		var req DomWaitRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = r.Body.Close()
		if req.Selector != "#x" || req.State != "visible" || req.TimeoutMS != 123 {
			t.Fatalf("wait=%#v", req)
		}
		_ = json.NewEncoder(w).Encode(DomWaitResponse{OK: true, State: req.State})
	})

	h.Mux.HandleFunc("/screenshot", func(w http.ResponseWriter, r *http.Request) {
		var req ScreenshotRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = r.Body.Close()
		if req.Format != "png" {
			t.Fatalf("screenshot format=%q", req.Format)
		}
		_ = json.NewEncoder(w).Encode(ScreenshotResponse{Format: "png", Base64: ""})
	})

	h.Mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(StopResponse{OK: true})
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

	if _, err := c.Goto(ctx, "/yolo"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Eval(ctx, "1+1"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DomAll(ctx, "li", "text"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DomAttr(ctx, "#x", "data-x"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DomClick(ctx, "#btn"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DomType(ctx, "#in", "hello", true); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DomWait(ctx, "#x", "visible", 123); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Screenshot(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Stop(ctx); err != nil {
		t.Fatal(err)
	}
}
