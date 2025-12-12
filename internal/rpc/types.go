package rpc

type StatusResponse struct {
	Running       bool   `json:"running"`
	BrowserAlive  bool   `json:"browser_alive"`
	PID           int    `json:"pid,omitempty"`
	Dir           string `json:"dir,omitempty"`
	HTTPAddr      string `json:"http_addr,omitempty"`
	HTTPPort      int    `json:"http_port,omitempty"`
	CurrentURL    string `json:"current_url,omitempty"`
	Title         string `json:"title,omitempty"`
	Headless      bool   `json:"headless,omitempty"`
	BrowserPID    int    `json:"browser_pid,omitempty"`
	DevToolsPort  int    `json:"devtools_port,omitempty"`
	DevToolsWSURL string `json:"devtools_ws_url,omitempty"`
	BrowserBinary string `json:"browser_bin,omitempty"`
	Error         string `json:"error,omitempty"`
}

type GotoRequest struct {
	URL string `json:"url"`
}

type GotoResponse struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

type EvalRequest struct {
	Expression string `json:"expression"`
}

type EvalResponse struct {
	Value any `json:"value"`
}

type ReloadResponse struct {
	OK bool `json:"ok"`
}

type DomRequest struct {
	Selector string `json:"selector"`
	Mode     string `json:"mode"` // "outer_html" | "text"
}

type DomResponse struct {
	Selector string `json:"selector"`
	Mode     string `json:"mode"`
	Value    string `json:"value"`
}

type DomAllRequest struct {
	Selector string `json:"selector"`
	Mode     string `json:"mode"` // "outer_html" | "text"
}

type DomAllResponse struct {
	Selector string   `json:"selector"`
	Mode     string   `json:"mode"`
	Values   []string `json:"values"`
}

type DomAttrRequest struct {
	Selector string `json:"selector"`
	Name     string `json:"name"`
}

type DomAttrResponse struct {
	Selector string  `json:"selector"`
	Name     string  `json:"name"`
	Value    *string `json:"value"`
}

type DomClickRequest struct {
	Selector string `json:"selector"`
}

type DomClickResponse struct {
	OK bool `json:"ok"`
}

type DomTypeRequest struct {
	Selector string `json:"selector"`
	Text     string `json:"text"`
	Clear    bool   `json:"clear,omitempty"`
}

type DomTypeResponse struct {
	OK bool `json:"ok"`
}

type DomWaitRequest struct {
	Selector  string `json:"selector"`
	State     string `json:"state"`      // "visible" | "hidden" | "ready" | "present" | "gone"
	TimeoutMS int    `json:"timeout_ms"` // 0 => default
}

type DomWaitResponse struct {
	OK    bool   `json:"ok"`
	State string `json:"state"`
}

type ScreenshotRequest struct {
	Selector string `json:"selector,omitempty"`
	Format   string `json:"format,omitempty"` // "png" only for now
}

type ScreenshotResponse struct {
	Format string `json:"format"`
	Base64 string `json:"base64"`
}

type StopResponse struct {
	OK bool `json:"ok"`
}
