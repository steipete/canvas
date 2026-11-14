package daemon

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestRunKeepsSocketWhenHTTPPortIsTaken(t *testing.T) {
	stateDir := t.TempDir()
	socketPath := filepath.Join(stateDir, "canvas.sock")
	if err := os.WriteFile(socketPath, []byte("live"), 0o600); err != nil {
		t.Fatal(err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	port := ln.Addr().(*net.TCPAddr).Port

	err = Run(Config{
		StateDir: stateDir,
		ServeDir: t.TempDir(),
		HTTPPort: port,
	})
	if err == nil {
		t.Fatal("expected Run to fail while the HTTP listener is still open")
	}
	if _, statErr := os.Stat(socketPath); statErr != nil {
		t.Fatalf("socket removed while %s is still bound: %v (run: %v)", ln.Addr(), statErr, err)
	}
}
