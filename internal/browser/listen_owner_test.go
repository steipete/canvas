package browser

import (
	"net"
	"os"
	"os/exec"
	"testing"
)

func TestProcNetListenInode(t *testing.T) {
	listen := "0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000 1000 0 123456 1 0000000000000000 100 0 0 10 0"
	inode, ok := procNetListenInode(listen, 8080)
	if !ok || inode != "123456" {
		t.Fatalf("inode=%q ok=%v", inode, ok)
	}
	if _, ok := procNetListenInode(listen, 80); ok {
		t.Fatal("matched a different port")
	}
	established := "1: 0100007F:1F90 0100007F:C3A0 01 00000000:00000000 00:00000000 00000000 1000 0 999 1 0000000000000000 100 0 0 10 0"
	if _, ok := procNetListenInode(established, 8080); ok {
		t.Fatal("matched a connected socket")
	}
}

func TestListenerOwnedByThisProcess(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	port := ln.Addr().(*net.TCPAddr).Port

	if !listenerOwnedBy(port, os.Getpid()) {
		t.Fatalf("port %d is not owned by pid %d", port, os.Getpid())
	}

	cmd := exec.Command("/bin/sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	if listenerOwnedBy(port, cmd.Process.Pid) {
		t.Fatalf("port %d was attributed to sleep pid %d", port, cmd.Process.Pid)
	}
}
