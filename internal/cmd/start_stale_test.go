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

func TestStartCommand_StaleSession_StartsInsteadOfReportingZero(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	if err := state.Save(stateDir, state.Session{
		PID:        1,
		SocketPath: shortSocketPath(t),
		Token:      "stale",
	}); err != nil {
		t.Fatal(err)
	}

	statusCmd := newStatusCmd(&rootFlags{})
	var statusBuf bytes.Buffer
	restore, err := captureStdout(&statusBuf)
	if err != nil {
		t.Fatal(err)
	}
	if err := statusCmd.Execute(); err != nil {
		_ = restore()
		t.Fatal(err)
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}
	if statusBuf.String() != "not running\n" {
		t.Fatalf("status = %q, want not running", statusBuf.String())
	}

	newSocket, shutdown := startUnixRPCServer(t, "token456", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StatusResponse{
				Running:  true,
				HTTPAddr: "127.0.0.1",
				HTTPPort: 2222,
				Dir:      "/tmp/new",
			})
		})
	})
	t.Cleanup(shutdown)

	spawned := false
	oldSpawn := spawnDaemon
	t.Cleanup(func() { spawnDaemon = oldSpawn })
	spawnDaemon = func(bin string, args []string, logFile *os.File) error {
		spawned = true
		return state.Save(stateDir, state.Session{PID: 2, SocketPath: newSocket, Token: "token456"})
	}

	cmd := newStartCmd(&rootFlags{})
	cmd.SetArgs([]string{"--dir", t.TempDir()})
	var buf bytes.Buffer
	restore, err = captureStdout(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Execute(); err != nil {
		_ = restore()
		t.Fatalf("start: %v; output %q", err, buf.String())
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}

	if !spawned {
		t.Fatalf("start did not spawn a daemon; output %q", buf.String())
	}
	got := buf.String()
	if strings.Contains(got, ":0/") {
		t.Fatalf("reported a zero status as running: %q", got)
	}
	if !strings.Contains(got, "http://127.0.0.1:2222/") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestStartCommand_StaleSession_JSONStarts(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	if err := state.Save(stateDir, state.Session{
		PID:        1,
		SocketPath: shortSocketPath(t),
		Token:      "stale",
	}); err != nil {
		t.Fatal(err)
	}

	newSocket, shutdown := startUnixRPCServer(t, "token456", func(mux *http.ServeMux) {
		mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(rpc.StatusResponse{
				Running:  true,
				HTTPAddr: "127.0.0.1",
				HTTPPort: 2222,
				Dir:      "/tmp/new",
			})
		})
	})
	t.Cleanup(shutdown)

	spawned := false
	oldSpawn := spawnDaemon
	t.Cleanup(func() { spawnDaemon = oldSpawn })
	spawnDaemon = func(bin string, args []string, logFile *os.File) error {
		spawned = true
		return state.Save(stateDir, state.Session{PID: 2, SocketPath: newSocket, Token: "token456"})
	}

	cmd := newStartCmd(&rootFlags{jsonOutput: true})
	cmd.SetArgs([]string{"--dir", t.TempDir()})
	var buf bytes.Buffer
	restore, err := captureStdout(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Execute(); err != nil {
		_ = restore()
		t.Fatalf("start: %v; output %q", err, buf.String())
	}
	if err := restore(); err != nil {
		t.Fatal(err)
	}
	if !spawned {
		t.Fatalf("start did not spawn a daemon; output %q", buf.String())
	}

	var st rpc.StatusResponse
	if err := json.Unmarshal(buf.Bytes(), &st); err != nil {
		t.Fatalf("decode %q: %v", buf.String(), err)
	}
	if !st.Running || st.HTTPPort != 2222 {
		t.Fatalf("status = %+v", st)
	}
}
