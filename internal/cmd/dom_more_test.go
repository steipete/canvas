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

func TestDomAllCommand_JSON(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	socketPath := stateDir + "/rpc.sock"
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	h := rpc.NewHandler("token123")
	h.Mux.HandleFunc("/dom/all", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rpc.DomAllResponse{
			Selector: "#x",
			Mode:     "text",
			Values:   []string{"a", "b"},
		})
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

	flags := &rootFlags{jsonOutput: true}
	cmd := newDomCmd(flags)
	cmd.SetArgs([]string{"all", "#x"})
	_ = cmd.Flags().Set("mode", "text")

	var buf bytes.Buffer
	restore, err := captureStdout(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Execute(); err != nil {
		_ = restore()
		t.Fatal(err)
	}
	_ = restore()

	var out rpc.DomAllResponse
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v output=%q", err, buf.String())
	}
	if len(out.Values) != 2 || out.Values[0] != "a" || out.Values[1] != "b" {
		t.Fatalf("unexpected values: %#v", out.Values)
	}
}

func TestDomClickCommand_TextOK(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	socketPath := stateDir + "/rpc.sock"
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	h := rpc.NewHandler("token123")
	h.Mux.HandleFunc("/dom/click", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rpc.DomClickResponse{OK: true})
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

	flags := &rootFlags{jsonOutput: false}
	cmd := newDomCmd(flags)
	cmd.SetArgs([]string{"click", "#btn"})

	var buf bytes.Buffer
	restore, err := captureStdout(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Execute(); err != nil {
		_ = restore()
		t.Fatal(err)
	}
	_ = restore()

	if got := buf.String(); got != "ok\n" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestDomAttrCommand_JSONNull(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	socketPath := stateDir + "/rpc.sock"
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	h := rpc.NewHandler("token123")
	h.Mux.HandleFunc("/dom/attr", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rpc.DomAttrResponse{Selector: "#x", Name: "data-x", Value: nil})
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

	flags := &rootFlags{jsonOutput: true}
	cmd := newDomCmd(flags)
	cmd.SetArgs([]string{"attr", "#x", "data-x"})

	var buf bytes.Buffer
	restore, err := captureStdout(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Execute(); err != nil {
		_ = restore()
		t.Fatal(err)
	}
	_ = restore()

	var out rpc.DomAttrResponse
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid json: %v output=%q", err, buf.String())
	}
	if out.Value != nil {
		t.Fatalf("expected null value, got: %#v", *out.Value)
	}
}

func TestDomWaitCommand_TextOK(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	socketPath := stateDir + "/rpc.sock"
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	h := rpc.NewHandler("token123")
	h.Mux.HandleFunc("/dom/wait", func(w http.ResponseWriter, r *http.Request) {
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

	flags := &rootFlags{jsonOutput: false}
	cmd := newDomCmd(flags)
	cmd.SetArgs([]string{"wait", "#x", "--timeout", "1s"})

	var buf bytes.Buffer
	restore, err := captureStdout(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Execute(); err != nil {
		_ = restore()
		t.Fatal(err)
	}
	_ = restore()

	if got := buf.String(); got != "ok\n" {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestDomTypeCommand_TextOK(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	socketPath := stateDir + "/rpc.sock"
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	h := rpc.NewHandler("token123")
	h.Mux.HandleFunc("/dom/type", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rpc.DomTypeResponse{OK: true})
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

	flags := &rootFlags{jsonOutput: false}
	cmd := newDomCmd(flags)
	cmd.SetArgs([]string{"type", "#x", "hello", "--clear"})

	var buf bytes.Buffer
	restore, err := captureStdout(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if err := cmd.Execute(); err != nil {
		_ = restore()
		t.Fatal(err)
	}
	_ = restore()

	if got := buf.String(); got != "ok\n" {
		t.Fatalf("unexpected output: %q", got)
	}
}
