package browser

import (
	"context"
	"encoding/json"
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
	type version struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}

	url := fmt.Sprintf("http://127.0.0.1:%d/json/version", port)
	client := &http.Client{Timeout: 2 * time.Second}

	deadline, hasDeadline := ctx.Deadline()
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err == nil && resp != nil {
			var v version
			decErr := json.NewDecoder(resp.Body).Decode(&v)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK && decErr == nil && v.WebSocketDebuggerURL != "" {
				return v.WebSocketDebuggerURL, nil
			}
		}

		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if hasDeadline && time.Now().After(deadline) {
			return "", context.DeadlineExceeded
		}
		time.Sleep(100 * time.Millisecond)
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
