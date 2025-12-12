package daemon

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/steipete/canvas/internal/browser"
	"github.com/steipete/canvas/internal/rpc"
	"github.com/steipete/canvas/internal/state"
	"github.com/steipete/canvas/internal/watch"
	"github.com/steipete/canvas/internal/web"
)

func Run(cfg Config) error {
	if cfg.StateDir == "" {
		return errors.New("missing state dir")
	}
	if cfg.ServeDir == "" {
		return errors.New("missing serve dir")
	}

	if err := os.MkdirAll(cfg.StateDir, 0o700); err != nil {
		return err
	}

	socketPath := filepath.Join(cfg.StateDir, "canvas.sock")
	_ = os.Remove(socketPath)

	token, err := randomToken(16)
	if err != nil {
		return err
	}

	// HTTP server for content.
	staticHandler, err := web.NewStaticHandler(cfg.ServeDir)
	if err != nil {
		return err
	}
	httpLn, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.HTTPPort))
	if err != nil {
		return fmt.Errorf("listen http: %w", err)
	}
	defer httpLn.Close()

	httpSrv := &http.Server{
		Handler: staticHandler,
	}

	go func() {
		_ = httpSrv.Serve(httpLn)
	}()

	actualPort := httpLn.Addr().(*net.TCPAddr).Port
	baseURL := fmt.Sprintf("http://127.0.0.1:%d/", actualPort)

	// Browser controller.
	profileDir := filepath.Join(cfg.StateDir, "chrome-profile")
	_ = os.RemoveAll(profileDir)
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return err
	}

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	controller, err := browser.New(rootCtx, browser.Options{
		BrowserBin:   cfg.BrowserBin,
		Headless:     cfg.Headless,
		UserDataDir:  profileDir,
		DevToolsPort: cfg.DevToolsPort,
	})
	if err != nil {
		_ = httpSrv.Shutdown(context.Background())
		return fmt.Errorf("launch browser: %w", err)
	}
	defer controller.Close()

	if _, _, err := controller.Navigate(rootCtx, baseURL); err != nil {
		return fmt.Errorf("navigate %s: %w", baseURL, err)
	}

	// RPC server.
	rpch := rpc.NewHandler(token)
	stopCh := make(chan struct{})
	var stopOnce sync.Once

	rpch.Mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		loc, _ := controller.Location(r.Context())
		title, _ := controller.Title(r.Context())
		out := rpc.StatusResponse{
			Running:       true,
			BrowserAlive:  controller.Alive(r.Context()),
			PID:           os.Getpid(),
			Dir:           cfg.ServeDir,
			HTTPAddr:      "127.0.0.1",
			HTTPPort:      actualPort,
			CurrentURL:    loc,
			Title:         title,
			Headless:      cfg.Headless,
			BrowserPID:    controller.BrowserPID(),
			DevToolsPort:  controller.DevToolsPort(),
			DevToolsWSURL: controller.DevToolsWSURL(),
			BrowserBinary: controller.BrowserBinary(),
		}
		rpcWriteJSON(w, http.StatusOK, out)
	})

	rpch.Mux.HandleFunc("/goto", func(w http.ResponseWriter, r *http.Request) {
		var req rpc.GotoRequest
		if err := rpcReadJSON(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		u := normalizeURL(baseURL, req.URL)
		loc, title, err := controller.Navigate(r.Context(), u)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rpcWriteJSON(w, http.StatusOK, rpc.GotoResponse{URL: loc, Title: title})
	})

	rpch.Mux.HandleFunc("/eval", func(w http.ResponseWriter, r *http.Request) {
		var req rpc.EvalRequest
		if err := rpcReadJSON(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		val, err := controller.Eval(r.Context(), req.Expression)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rpcWriteJSON(w, http.StatusOK, rpc.EvalResponse{Value: val})
	})

	rpch.Mux.HandleFunc("/reload", func(w http.ResponseWriter, r *http.Request) {
		if err := controller.Reload(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rpcWriteJSON(w, http.StatusOK, rpc.ReloadResponse{OK: true})
	})

	rpch.Mux.HandleFunc("/dom", func(w http.ResponseWriter, r *http.Request) {
		var req rpc.DomRequest
		if err := rpcReadJSON(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mode := req.Mode
		if mode == "" {
			mode = "outer_html"
		}
		var val string
		var err error
		switch mode {
		case "outer_html":
			val, err = controller.OuterHTML(r.Context(), req.Selector)
		case "text":
			val, err = controller.Text(r.Context(), req.Selector)
		default:
			http.Error(w, "unknown mode", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rpcWriteJSON(w, http.StatusOK, rpc.DomResponse{Selector: req.Selector, Mode: mode, Value: val})
	})

	rpch.Mux.HandleFunc("/screenshot", func(w http.ResponseWriter, r *http.Request) {
		var req rpc.ScreenshotRequest
		if err := rpcReadJSON(r, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		buf, err := controller.Screenshot(r.Context(), req.Selector)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rpcWriteJSON(w, http.StatusOK, rpc.ScreenshotResponse{
			Format: "png",
			Base64: base64.StdEncoding.EncodeToString(buf),
		})
	})

	rpch.Mux.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		rpcWriteJSON(w, http.StatusOK, rpc.StopResponse{OK: true})
		go func() {
			time.Sleep(100 * time.Millisecond)
			stopOnce.Do(func() { close(stopCh) })
		}()
	})

	unixLn, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("listen rpc: %w", err)
	}
	if err := os.Chmod(socketPath, 0o600); err != nil {
		_ = unixLn.Close()
		return err
	}
	defer unixLn.Close()

	rpcSrv := &http.Server{Handler: rpch}
	go func() { _ = rpcSrv.Serve(unixLn) }()

	sess := state.Session{
		PID:           os.Getpid(),
		StartedAt:     time.Now(),
		Dir:           cfg.ServeDir,
		HTTPAddr:      "127.0.0.1",
		HTTPPort:      actualPort,
		SocketPath:    socketPath,
		Token:         token,
		Headless:      cfg.Headless,
		BrowserPID:    controller.BrowserPID(),
		DevToolsPort:  controller.DevToolsPort(),
		DevToolsWSURL: controller.DevToolsWSURL(),
		BrowserBin:    controller.BrowserBinary(),
	}
	if err := state.Save(cfg.StateDir, sess); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	defer func() {
		_ = state.Remove(cfg.StateDir)
		_ = os.Remove(socketPath)
		if cfg.TempDir {
			_ = os.RemoveAll(cfg.ServeDir)
		}
	}()

	// File watcher for auto-reload.
	if cfg.Watch {
		go func() {
			_ = watch.WatchRecursive(rootCtx, cfg.ServeDir, watch.Options{Debounce: 250 * time.Millisecond}, func() {
				_ = controller.Reload(context.Background())
			})
		}()
	}

	// Handle signals.
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-stopCh:
	case <-sigCh:
	case <-rootCtx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	_ = rpcSrv.Shutdown(shutdownCtx)
	_ = httpSrv.Shutdown(shutdownCtx)
	_ = controller.Close()

	// Unblock watcher goroutine.
	cancel()

	return nil
}

func normalizeURL(baseURL, in string) string {
	s := strings.TrimSpace(in)
	if s == "" {
		return baseURL
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	return strings.TrimRight(baseURL, "/") + s
}

func rpcWriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func rpcReadJSON(r *http.Request, out any) error {
	if r.Body == nil {
		return errors.New("missing body")
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(out)
}
