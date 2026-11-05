package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// DevToolsWebSocketURL returns the browser-level DevTools websocket URL by querying
// the remote debugging HTTP endpoint at /json/version.
func DevToolsWebSocketURL(port int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return devToolsWebSocketURL(ctx, port)
}

func devToolsWebSocketURL(ctx context.Context, port int) (string, error) {
	deadline, hasDeadline := ctx.Deadline()
	for {
		ws, err := readDevToolsWebSocketURL(ctx, port)
		if err == nil && ws != "" {
			return ws, nil
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if hasDeadline && time.Now().After(deadline) {
			return "", context.DeadlineExceeded
		}
		if !waitCtx(ctx, 100*time.Millisecond) {
			if ctx.Err() != nil {
				return "", ctx.Err()
			}
			return "", context.DeadlineExceeded
		}
	}
}

// devToolsWebSocketURLForBrowser waits until the browser process pid owns
// the DevTools port. The first /json/version on that port can still be a
// browser this process did not start.
func devToolsWebSocketURLForBrowser(ctx context.Context, port, pid int) (string, error) {
	if pid <= 0 {
		return "", errors.New("missing browser pid")
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
	}

	var foreign int
	for {
		if err := ctx.Err(); err != nil {
			if foreign > 0 {
				return "", fmt.Errorf("devtools port %d belongs to pid %d, not browser %d: %w", port, foreign, pid, err)
			}
			return "", fmt.Errorf("browser %d did not own devtools port %d: %w", pid, port, err)
		}
		if browserProcessExited(pid) {
			return "", fmt.Errorf("browser %d exited before owning devtools port %d", pid, port)
		}
		owned, other := listenerOwnership(port, pid)
		if owned {
			ws, err := readDevToolsWebSocketURL(ctx, port)
			if err == nil && ws != "" && listenerOwnedBy(port, pid) {
				return ws, nil
			}
		} else if other > 0 {
			foreign = other
		}
		waitCtx(ctx, 50*time.Millisecond)
	}
}

func readDevToolsWebSocketURL(ctx context.Context, port int) (string, error) {
	type version struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("devtools version: %s", resp.Status)
	}
	var v version
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", err
	}
	if v.WebSocketDebuggerURL == "" {
		return "", errors.New("empty devtools websocket url")
	}
	return v.WebSocketDebuggerURL, nil
}

func waitCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type DevToolsTarget struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func DevToolsTargets(port int) ([]DevToolsTarget, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return devToolsTargets(ctx, port)
}

func devToolsTargets(ctx context.Context, port int) ([]DevToolsTarget, error) {
	url := fmt.Sprintf("http://127.0.0.1:%d/json/list", port)
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("devtools targets: %s", resp.Status)
	}
	var out []DevToolsTarget
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
