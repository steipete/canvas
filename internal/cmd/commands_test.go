package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/steipete/canvas/internal/rpc"
	"github.com/steipete/canvas/internal/state"
)

func TestStatusCommand_NotRunning(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	flags := &rootFlags{jsonOutput: false}
	cmd := newStatusCmd(flags)

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

	if buf.String() != "not running\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestGotoCommand_Text(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	socketPath, shutdown := startUnixRPCServer(t, "token123", func(mux *http.ServeMux) {
		mux.HandleFunc("/goto", func(w http.ResponseWriter, r *http.Request) {
			var req rpc.GotoRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			_ = r.Body.Close()
			_ = json.NewEncoder(w).Encode(rpc.GotoResponse{URL: "http://127.0.0.1:1/yolo"})
		})
	})
	t.Cleanup(shutdown)

	if err := state.Save(stateDir, state.Session{PID: 1, SocketPath: socketPath, Token: "token123"}); err != nil {
		t.Fatal(err)
	}

	flags := &rootFlags{jsonOutput: false}
	cmd := newGotoCmd(flags)
	cmd.SetArgs([]string{"/yolo"})

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
	if buf.String() != "http://127.0.0.1:1/yolo\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestEvalCommand_Text_String(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	socketPath, shutdown := startUnixRPCServer(t, "token123", func(mux *http.ServeMux) {
		mux.HandleFunc("/eval", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.EvalResponse{Value: "hello"})
		})
	})
	t.Cleanup(shutdown)

	if err := state.Save(stateDir, state.Session{PID: 1, SocketPath: socketPath, Token: "token123"}); err != nil {
		t.Fatal(err)
	}

	flags := &rootFlags{jsonOutput: false}
	cmd := newEvalCmd(flags)
	cmd.SetArgs([]string{"1+1"})

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
	if buf.String() != "hello\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestReloadCommand_Text(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	socketPath, shutdown := startUnixRPCServer(t, "token123", func(mux *http.ServeMux) {
		mux.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.ReloadResponse{OK: true})
		})
	})
	t.Cleanup(shutdown)

	if err := state.Save(stateDir, state.Session{PID: 1, SocketPath: socketPath, Token: "token123"}); err != nil {
		t.Fatal(err)
	}

	flags := &rootFlags{jsonOutput: false}
	cmd := newReloadCmd(flags)

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
	if buf.String() != "ok\n" {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}

func TestScreenshotCommand_WritesFile(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	want := []byte{0x89, 0x50, 0x4e, 0x47}
	socketPath, shutdown := startUnixRPCServer(t, "token123", func(mux *http.ServeMux) {
		mux.HandleFunc("/screenshot", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.ScreenshotResponse{
				Format: "png",
				Base64: base64.StdEncoding.EncodeToString(want),
			})
		})
	})
	t.Cleanup(shutdown)

	if err := state.Save(stateDir, state.Session{PID: 1, SocketPath: socketPath, Token: "token123"}); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(t.TempDir(), "shot.png")
	flags := &rootFlags{jsonOutput: false}
	cmd := newScreenshotCmd(flags)
	cmd.SetArgs([]string{"--out", outPath})

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

	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, want) {
		t.Fatalf("file bytes mismatch: %v != %v", b, want)
	}
}

func startUnixRPCServer(t *testing.T, token string, register func(mux *http.ServeMux)) (socketPath string, shutdown func()) {
	t.Helper()

	socketPath = shortSocketPath(t)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}

	h := rpc.NewHandler(token)
	register(h.Mux)

	srv := &http.Server{Handler: h}
	go func() { _ = srv.Serve(ln) }()

	return socketPath, func() {
		_ = ln.Close()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
}
