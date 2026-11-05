package daemon

import (
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestStopReleasesHTTPListenerBeforeDelayedSignal(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	srv := &http.Server{Handler: http.NewServeMux()}
	go func() { _ = srv.Serve(ln) }()
	waitUntilDial(t, ln.Addr().String())

	stopCh := make(chan struct{})
	var once sync.Once
	start := time.Now()
	releaseHTTPServer(srv)
	if time.Since(start) >= 80*time.Millisecond {
		t.Fatalf("HTTP release waited out the stop delay: %s", time.Since(start))
	}

	probe, err := net.Listen("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("HTTP listener still held after stop: %v", err)
	}
	_ = probe.Close()

	scheduleDaemonStop(&once, stopCh)
	select {
	case <-stopCh:
		t.Fatal("process stop fired before the response delay")
	case <-time.After(20 * time.Millisecond):
	}
	select {
	case <-stopCh:
	case <-time.After(time.Second):
		t.Fatal("delayed stop signal was not delivered")
	}
}

func waitUntilDial(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			_ = conn.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
