package browser

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("CANVAS_FAKE_BROWSER") == "1" {
		os.Exit(runFakeBrowser())
	}
	os.Exit(m.Run())
}

func TestLaunchDoesNotAttachToForeignDevTools(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	foreignURL := fmt.Sprintf("ws://127.0.0.1:%d/devtools/browser/foreign", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"webSocketDebuggerUrl":"` + foreignURL + `"}`))
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	got, err := launch(ctx, LaunchOptions{
		BrowserBin:   writeSleepBrowser(t),
		DevToolsPort: port,
		StartURL:     "about:blank",
	})
	if err == nil {
		_ = terminateProcess(got.Cmd, time.Second)
		t.Fatalf("attached to DevTools port %d owned by another process: %s", port, got.DevToolsWS)
	}
	if got.DevToolsWS == foreignURL {
		t.Fatalf("returned foreign websocket %s", got.DevToolsWS)
	}
}

func TestLaunchAcceptsDevToolsOwnedByTheChild(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := launch(ctx, LaunchOptions{
		BrowserBin:   writeFakeBrowser(t),
		DevToolsPort: freePort(t),
		StartURL:     "about:blank",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = terminateProcess(got.Cmd, time.Second) })
	if !strings.Contains(got.DevToolsWS, "/devtools/browser/child") {
		t.Fatalf("websocket %q is not the child browser", got.DevToolsWS)
	}
	if got.Cmd == nil || got.Cmd.Process == nil {
		t.Fatal("missing browser process")
	}
}

func writeSleepBrowser(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sleep-browser")
	script := "#!/bin/sh\nexec sleep 30\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeFakeBrowser(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fake-browser")
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nexport CANVAS_FAKE_BROWSER=1\nexec " + shellQuote(bin) + " \"$@\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}

func runFakeBrowser() int {
	port := 0
	for _, arg := range os.Args[1:] {
		const prefix = "--remote-debugging-port="
		if strings.HasPrefix(arg, prefix) {
			parsed, err := strconv.Atoi(strings.TrimPrefix(arg, prefix))
			if err != nil || parsed <= 0 {
				fmt.Fprintln(os.Stderr, "bad devtools port")
				return 1
			}
			port = parsed
		}
	}
	if port <= 0 {
		fmt.Fprintln(os.Stderr, "missing devtools port")
		return 1
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/json/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"webSocketDebuggerUrl":"ws://127.0.0.1:%d/devtools/browser/child"}`, port)
	})
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	srv := &http.Server{Handler: mux}
	_ = srv.Serve(ln)
	return 0
}
