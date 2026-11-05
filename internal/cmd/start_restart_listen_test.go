package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/steipete/canvas/internal/rpc"
	"github.com/steipete/canvas/internal/state"
)

// The old /stop handler answers before it closes listeners. Hold a real
// TCP port across that answer so a replacement that binds immediately
// fails, which is what canvas start --restart does before the fix.
func TestStartRestartWaitsUntilOldHTTPListenerCloses(t *testing.T) {
	restartWhileListenerHeld(t, true, false)
}

func TestStartRestartWaitsUntilOldDevToolsListenerCloses(t *testing.T) {
	restartWhileListenerHeld(t, false, true)
}

func restartWhileListenerHeld(t *testing.T, holdHTTP, holdDevTools bool) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	addr := ln.Addr().String()

	var closeOnce sync.Once
	closeListener := func() { closeOnce.Do(func() { _ = ln.Close() }) }
	t.Cleanup(closeListener)
	releaseListener := func() {
		time.Sleep(300 * time.Millisecond)
		closeListener()
	}

	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	status := rpc.StatusResponse{
		Running:  true,
		HTTPAddr: "127.0.0.1",
		Dir:      "/tmp/old",
	}
	if holdHTTP {
		status.HTTPPort = port
	}
	if holdDevTools {
		status.DevToolsPort = port
	}

	oldSocket, shutdownOld := startUnixRPCServer(t, "token123", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(status)
		})
		mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StopResponse{OK: true})
			go releaseListener()
		})
	})
	t.Cleanup(shutdownOld)

	if err := state.Save(stateDir, state.Session{PID: 1, SocketPath: oldSocket, Token: "token123"}); err != nil {
		t.Fatal(err)
	}

	reported := port + 1
	if reported > 65535 {
		reported = port - 1
	}
	newSocket, shutdownNew := startUnixRPCServer(t, "token456", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StatusResponse{
				Running:  true,
				HTTPAddr: "127.0.0.1",
				HTTPPort: reported,
				Dir:      "/tmp/new",
			})
		})
	})
	t.Cleanup(shutdownNew)

	oldSpawn := spawnDaemon
	t.Cleanup(func() { spawnDaemon = oldSpawn })
	spawnDaemon = func(bin string, args []string, logFile *os.File) error {
		probe, listenErr := net.Listen("tcp", addr)
		if listenErr != nil {
			return fmt.Errorf("replacement started while %s was still bound: %w", addr, listenErr)
		}
		_ = probe.Close()
		return state.Save(stateDir, state.Session{PID: 2, SocketPath: newSocket, Token: "token456"})
	}

	flags := &rootFlags{jsonOutput: false}
	cmd := newStartCmd(flags)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	args := []string{"--restart"}
	if holdHTTP {
		args = append(args, "--port", strconv.Itoa(port))
	}
	if holdDevTools {
		args = append(args, "--devtools-port", strconv.Itoa(port))
	}
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}
