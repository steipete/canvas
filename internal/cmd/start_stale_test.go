package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

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

func TestStartCommand_StatusTimeout_KeepsSession(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("CANVAS_STATE_DIR", stateDir)

	socketPath := shortSocketPath(t)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	accepted := make(chan net.Conn, 4)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			select {
			case accepted <- conn:
			default:
				_ = conn.Close()
			}
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		for {
			select {
			case conn := <-accepted:
				_ = conn.Close()
			default:
				return
			}
		}
	})

	if err := state.Save(stateDir, state.Session{
		PID:        7,
		SocketPath: socketPath,
		Token:      "busy",
		Dir:        "/tmp/kept",
		HTTPPort:   9,
	}); err != nil {
		t.Fatal(err)
	}
	sessionPath := state.SessionPath(stateDir)
	before, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatal(err)
	}

	spawned := false
	oldSpawn := spawnDaemon
	t.Cleanup(func() { spawnDaemon = oldSpawn })
	spawnDaemon = func(bin string, args []string, logFile *os.File) error {
		spawned = true
		return errors.New("spawnDaemon should not be called when status times out")
	}

	cmd := newStartCmd(&rootFlags{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetArgs([]string{"--dir", t.TempDir()})
	var buf bytes.Buffer
	restore, err := captureStdout(&buf)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	err = cmd.Execute()
	elapsed := time.Since(started)
	if restoreErr := restore(); restoreErr != nil {
		t.Fatal(restoreErr)
	}

	if err == nil || !strings.Contains(err.Error(), "cannot query the existing daemon") {
		t.Fatalf("error = %v, output %q", err, buf.String())
	}
	if spawned {
		t.Fatal("start spawned a daemon")
	}
	if elapsed < 1500*time.Millisecond {
		t.Fatalf("status returned in %s, want the 2s deadline", elapsed)
	}
	after, readErr := os.ReadFile(sessionPath)
	if readErr != nil {
		t.Fatalf("session file: %v", readErr)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("session.json changed:\n%s", after)
	}
	if buf.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", buf.String())
	}
}

func TestStatusEndpointDeadConservative(t *testing.T) {
	for _, err := range []error{syscall.EACCES, syscall.EMFILE, contextDeadlineError{}} {
		if statusEndpointDead(err) {
			t.Fatalf("inconclusive error classified dead: %v", err)
		}
	}
	for _, err := range []error{syscall.ENOENT, syscall.ECONNREFUSED} {
		if !statusEndpointDead(err) {
			t.Fatalf("dead endpoint not recognized: %v", err)
		}
	}
}

type contextDeadlineError struct{}

func (contextDeadlineError) Error() string { return "deadline exceeded" }
