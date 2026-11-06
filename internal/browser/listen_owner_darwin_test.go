//go:build darwin

package browser

import (
	"net"
	"os"
	"os/exec"
	"testing"
)

func TestListeningPIDsWithMinimalPATH(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	if _, err := exec.LookPath("lsof"); err == nil {
		t.Fatal("lsof is present on PATH=/usr/bin:/bin")
	}
	got := lsofPath()
	if got != "/usr/sbin/lsof" {
		t.Fatalf("lsof path = %q, want /usr/sbin/lsof", got)
	}
	t.Logf("PATH=%s lsof=%s", os.Getenv("PATH"), got)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	t.Cleanup(func() { _ = ln.Close() })

	pids, err := listeningPIDs(port)
	if err != nil {
		t.Fatalf("listeningPIDs: %v", err)
	}
	t.Logf("127.0.0.1:%d listening pids=%v this=%d", port, pids, os.Getpid())
	if !pidListContains(pids, os.Getpid()) {
		t.Fatalf("port %d pids=%v, want %d", port, pids, os.Getpid())
	}

	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	pids, err = listeningPIDs(port)
	t.Logf("closed port %d pids=%v err=%v", port, pids, err)
	if err != nil {
		t.Fatalf("closed port returned %v; lsof exit 1 should mean nobody is listening", err)
	}
	if pidListContains(pids, os.Getpid()) {
		t.Fatalf("closed port %d still lists pid %d in %v", port, os.Getpid(), pids)
	}
}

func pidListContains(pids []int, want int) bool {
	for _, pid := range pids {
		if pid == want {
			return true
		}
	}
	return false
}
