package daemon

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/steipete/canvas/internal/state"
)

func TestReleaseKeepsTheReplacementSessionAndSocket(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "canvas.sock")
	if err := os.WriteFile(socketPath, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := os.Stat(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(socketPath, filepath.Join(dir, "old.sock")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(socketPath, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	serveDir := filepath.Join(dir, "serve")
	if err := os.Mkdir(serveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := state.Save(dir, state.Session{PID: 22, Dir: serveDir, SocketPath: socketPath}); err != nil {
		t.Fatal(err)
	}

	releaseDaemonState(dir, socketPath, 11, original, true, serveDir)

	current, err := state.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if current.PID != 22 {
		t.Fatalf("session pid = %d", current.PID)
	}
	if _, err := os.Stat(socketPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(serveDir); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseRemovesOwnSessionAndSocket(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "canvas.sock")
	if err := os.WriteFile(socketPath, []byte("own"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	serveDir := filepath.Join(dir, "serve")
	if err := os.Mkdir(serveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := state.Save(dir, state.Session{PID: 11, Dir: serveDir, SocketPath: socketPath}); err != nil {
		t.Fatal(err)
	}

	releaseDaemonState(dir, socketPath, 11, info, true, serveDir)

	if _, err := state.Load(dir); !os.IsNotExist(err) {
		t.Fatalf("session err = %v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("socket err = %v", err)
	}
	if _, err := os.Stat(serveDir); !os.IsNotExist(err) {
		t.Fatalf("serve dir err = %v", err)
	}
}

func TestOldListenerCloseKeepsReplacementSocket(t *testing.T) {
	dir, err := os.MkdirTemp("", "cs-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socketPath := filepath.Join(dir, "s")
	oldListener, oldInfo, err := listenRPCSocket(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = oldListener.Close() })
	if err := os.Remove(socketPath); err != nil {
		t.Fatal(err)
	}
	newListener, _, err := listenRPCSocket(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = newListener.Close() })
	if err := state.Save(dir, state.Session{PID: 22, SocketPath: socketPath}); err != nil {
		t.Fatal(err)
	}
	releaseDaemonState(dir, socketPath, 11, oldInfo, false, "")
	_ = oldListener.Close()
	connection, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("replacement socket disappeared: %v", err)
	}
	_ = connection.Close()
}

func TestReleaseRemovesUnusedOldTempDirectory(t *testing.T) {
	dir := t.TempDir()
	oldDir := filepath.Join(dir, "old")
	newDir := filepath.Join(dir, "new")
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := state.Save(dir, state.Session{PID: 22, Dir: newDir}); err != nil {
		t.Fatal(err)
	}
	releaseDaemonState(dir, "", 11, nil, true, oldDir)
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Fatalf("old temp dir remains: %v", err)
	}
	if _, err := os.Stat(newDir); err != nil {
		t.Fatalf("new temp dir removed: %v", err)
	}
}
