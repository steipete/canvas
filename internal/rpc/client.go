package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

func NewUnixClient(socketPath, token string) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := &net.Dialer{Timeout: 3 * time.Second}
			return d.DialContext(ctx, "unix", socketPath)
		},
	}
	return &Client{
		baseURL: "http://unix",
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		token: token,
	}
}

func (c *Client) doJSON(ctx context.Context, method, path string, reqBody any, out any) error {
	var body *bytes.Reader
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s failed: %s", method, path, resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) Status(ctx context.Context) (StatusResponse, error) {
	var out StatusResponse
	err := c.doJSON(ctx, http.MethodGet, "/status", nil, &out)
	return out, err
}

func (c *Client) Goto(ctx context.Context, url string) (GotoResponse, error) {
	var out GotoResponse
	err := c.doJSON(ctx, http.MethodPost, "/goto", GotoRequest{URL: url}, &out)
	return out, err
}

func (c *Client) Eval(ctx context.Context, expr string) (EvalResponse, error) {
	var out EvalResponse
	err := c.doJSON(ctx, http.MethodPost, "/eval", EvalRequest{Expression: expr}, &out)
	return out, err
}

func (c *Client) Reload(ctx context.Context) (ReloadResponse, error) {
	var out ReloadResponse
	err := c.doJSON(ctx, http.MethodPost, "/reload", nil, &out)
	return out, err
}

func (c *Client) Dom(ctx context.Context, selector, mode string) (DomResponse, error) {
	var out DomResponse
	err := c.doJSON(ctx, http.MethodPost, "/dom", DomRequest{Selector: selector, Mode: mode}, &out)
	return out, err
}

func (c *Client) DomAll(ctx context.Context, selector, mode string) (DomAllResponse, error) {
	var out DomAllResponse
	err := c.doJSON(ctx, http.MethodPost, "/dom/all", DomAllRequest{Selector: selector, Mode: mode}, &out)
	return out, err
}

func (c *Client) DomAttr(ctx context.Context, selector, name string) (DomAttrResponse, error) {
	var out DomAttrResponse
	err := c.doJSON(ctx, http.MethodPost, "/dom/attr", DomAttrRequest{Selector: selector, Name: name}, &out)
	return out, err
}

func (c *Client) DomClick(ctx context.Context, selector string) (DomClickResponse, error) {
	var out DomClickResponse
	err := c.doJSON(ctx, http.MethodPost, "/dom/click", DomClickRequest{Selector: selector}, &out)
	return out, err
}

func (c *Client) DomType(ctx context.Context, selector, text string, clear bool) (DomTypeResponse, error) {
	var out DomTypeResponse
	err := c.doJSON(ctx, http.MethodPost, "/dom/type", DomTypeRequest{Selector: selector, Text: text, Clear: clear}, &out)
	return out, err
}

func (c *Client) DomWait(ctx context.Context, selector, state string, timeoutMS int) (DomWaitResponse, error) {
	var out DomWaitResponse
	err := c.doJSON(ctx, http.MethodPost, "/dom/wait", DomWaitRequest{Selector: selector, State: state, TimeoutMS: timeoutMS}, &out)
	return out, err
}

func (c *Client) Screenshot(ctx context.Context, selector string) (ScreenshotResponse, error) {
	var out ScreenshotResponse
	err := c.doJSON(ctx, http.MethodPost, "/screenshot", ScreenshotRequest{Selector: selector, Format: "png"}, &out)
	return out, err
}

func (c *Client) Stop(ctx context.Context) (StopResponse, error) {
	var out StopResponse
	err := c.doJSON(ctx, http.MethodPost, "/stop", nil, &out)
	return out, err
}
