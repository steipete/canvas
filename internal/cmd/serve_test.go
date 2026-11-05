package cmd

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/steipete/canvas/internal/rpc"
	"github.com/steipete/canvas/internal/state"
)

func TestServeKeepsSessionWhenHTTPPortStaysBound(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)
	oldTimeout := httpPortReleaseTimeout
	httpPortReleaseTimeout = 200 * time.Millisecond
	t.Cleanup(func() { httpPortReleaseTimeout = oldTimeout })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	port := ln.Addr().(*net.TCPAddr).Port

	var stopCalled atomic.Bool
	socketPath, shutdown := startUnixRPCServer(t, "token123", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StatusResponse{
				Running:  true,
				PID:      4242,
				HTTPAddr: "127.0.0.1",
				HTTPPort: port,
				Dir:      "/tmp/old",
			})
		})
		mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
			stopCalled.Store(true)
			_ = json.NewEncoder(w).Encode(rpc.StopResponse{OK: true})
		})
	})
	t.Cleanup(shutdown)

	orig := state.Session{
		PID:        4242,
		Dir:        "/tmp/old",
		HTTPAddr:   "127.0.0.1",
		HTTPPort:   port,
		SocketPath: socketPath,
		Token:      "token123",
	}
	if err := state.Save(stateDir, orig); err != nil {
		t.Fatal(err)
	}

	cmd := newServeCmd(&rootFlags{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--port", strconv.Itoa(port), "--dir", t.TempDir()})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected serve to fail while the HTTP listener is still open")
	}

	got, loadErr := state.Load(stateDir)
	if loadErr != nil {
		t.Fatalf("session removed while %s is still bound: load %v; serve %v", ln.Addr(), loadErr, err)
	}
	if got.PID != orig.PID || got.Token != orig.Token || got.SocketPath != orig.SocketPath || got.HTTPPort != orig.HTTPPort {
		t.Fatalf("session = %+v, want pid %d token %s port %d", got, orig.PID, orig.Token, orig.HTTPPort)
	}
	if !stopCalled.Load() {
		t.Fatal("expected serve to stop the recorded daemon before replacing state")
	}

	probe, probeErr := net.Listen("tcp", ln.Addr().String())
	if probeErr == nil {
		_ = probe.Close()
		t.Fatal("original HTTP listener no longer holds the port")
	}
}

func TestServeKeepsSessionWhenStopFails(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	port := ln.Addr().(*net.TCPAddr).Port

	socketPath, shutdown := startUnixRPCServer(t, "token123", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StatusResponse{
				Running:  true,
				PID:      4242,
				HTTPAddr: "127.0.0.1",
				HTTPPort: port,
			})
		})
		mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "stop failed", http.StatusInternalServerError)
		})
	})
	t.Cleanup(shutdown)

	if err := state.Save(stateDir, state.Session{
		PID:        4242,
		HTTPAddr:   "127.0.0.1",
		HTTPPort:   port,
		SocketPath: socketPath,
		Token:      "token123",
	}); err != nil {
		t.Fatal(err)
	}

	cmd := newServeCmd(&rootFlags{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--port", strconv.Itoa(port), "--dir", t.TempDir()})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected serve to fail when stop fails")
	}
	if _, loadErr := state.Load(stateDir); loadErr != nil {
		t.Fatalf("session removed after stop failed: load %v; serve %v", loadErr, err)
	}
}

func TestServeLeavesRunningDaemonWhenOtherPortIsBusy(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	busy := ln.Addr().(*net.TCPAddr).Port
	recorded := 1
	if busy == recorded {
		recorded = 2
	}

	var stopCalled atomic.Bool
	socketPath, shutdown := startUnixRPCServer(t, "token123", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StatusResponse{
				Running:  true,
				PID:      4242,
				HTTPAddr: "127.0.0.1",
				HTTPPort: recorded,
			})
		})
		mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
			stopCalled.Store(true)
			_ = json.NewEncoder(w).Encode(rpc.StopResponse{OK: true})
		})
	})
	t.Cleanup(shutdown)

	if err := state.Save(stateDir, state.Session{
		PID:        4242,
		HTTPAddr:   "127.0.0.1",
		HTTPPort:   recorded,
		SocketPath: socketPath,
		Token:      "token123",
	}); err != nil {
		t.Fatal(err)
	}

	cmd := newServeCmd(&rootFlags{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--port", strconv.Itoa(busy), "--dir", t.TempDir()})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("expected serve to fail while the requested port is bound")
	}
	if _, loadErr := state.Load(stateDir); loadErr != nil {
		t.Fatalf("session removed while %s is still bound: load %v; serve %v", ln.Addr(), loadErr, err)
	}
	if stopCalled.Load() {
		t.Fatal("stopped the recorded daemon even though a different port is busy")
	}
}

func TestStopDaemonBeforeServeReturnsWhenListenerCloses(t *testing.T) {
	stateDir := t.TempDir()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	closed := make(chan struct{})
	socketPath, shutdown := startUnixRPCServer(t, "token123", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StatusResponse{
				Running:  true,
				PID:      7,
				HTTPAddr: "127.0.0.1",
				HTTPPort: port,
			})
		})
		mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
			_ = ln.Close()
			close(closed)
			_ = json.NewEncoder(w).Encode(rpc.StopResponse{OK: true})
		})
	})
	t.Cleanup(shutdown)
	t.Cleanup(func() { _ = ln.Close() })

	if err := state.Save(stateDir, state.Session{
		PID:        7,
		HTTPAddr:   "127.0.0.1",
		HTTPPort:   port,
		SocketPath: socketPath,
		Token:      "token123",
	}); err != nil {
		t.Fatal(err)
	}

	if err := stopDaemonBeforeServe(stateDir, port); err != nil {
		t.Fatal(err)
	}
	if _, err := state.Load(stateDir); err != nil {
		t.Fatalf("session removed before the caller replaces it: %v", err)
	}
	select {
	case <-closed:
	default:
		t.Fatal("stop was not called")
	}
}
