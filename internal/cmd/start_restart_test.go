package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/steipete/canvas/internal/rpc"
	"github.com/steipete/canvas/internal/state"
)

func TestStartCommand_AlreadyRunning_DoesNotSpawn(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	socketPath, shutdown := startUnixRPCServer(t, "token123", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StatusResponse{
				Running:  true,
				HTTPAddr: "127.0.0.1",
				HTTPPort: 1111,
				Dir:      "/tmp/dir",
			})
		})
	})
	t.Cleanup(shutdown)

	if err := state.Save(stateDir, state.Session{PID: 1, SocketPath: socketPath, Token: "token123"}); err != nil {
		t.Fatal(err)
	}

	oldSpawn := spawnDaemon
	t.Cleanup(func() { spawnDaemon = oldSpawn })
	spawnDaemon = func(bin string, args []string, logFile *os.File) error {
		t.Fatalf("spawnDaemon should not be called when already running")
		return nil
	}

	flags := &rootFlags{jsonOutput: false}
	cmd := newStartCmd(flags)

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

	if got := buf.String(); !strings.HasPrefix(got, "running:") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestStartCommand_Restart_StopsAndSpawns(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	stopCalled := false
	oldSocket, shutdownOld := startUnixRPCServer(t, "token123", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StatusResponse{
				Running:  true,
				HTTPAddr: "127.0.0.1",
				HTTPPort: 1111,
				Dir:      "/tmp/old",
			})
		})
		mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
			stopCalled = true
			_ = json.NewEncoder(w).Encode(rpc.StopResponse{OK: true})
		})
	})
	t.Cleanup(shutdownOld)

	if err := state.Save(stateDir, state.Session{PID: 1, SocketPath: oldSocket, Token: "token123"}); err != nil {
		t.Fatal(err)
	}

	newSocket, shutdownNew := startUnixRPCServer(t, "token456", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StatusResponse{
				Running:  true,
				HTTPAddr: "127.0.0.1",
				HTTPPort: 2222,
				Dir:      "/tmp/new",
			})
		})
	})
	t.Cleanup(shutdownNew)

	oldSpawn := spawnDaemon
	t.Cleanup(func() { spawnDaemon = oldSpawn })
	spawnDaemon = func(bin string, args []string, logFile *os.File) error {
		// Simulate the daemon writing its session file after "starting".
		return state.Save(stateDir, state.Session{PID: 2, SocketPath: newSocket, Token: "token456"})
	}

	flags := &rootFlags{jsonOutput: false}
	cmd := newStartCmd(flags)
	cmd.SetArgs([]string{"--restart"})

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

	if !stopCalled {
		t.Fatalf("expected restart to call /stop")
	}
	if got := buf.String(); !strings.HasPrefix(got, "running:") {
		t.Fatalf("unexpected output: %q", got)
	}
}
