package daemon

type Config struct {
	StateDir     string
	ServeDir     string
	HTTPPort     int
	DevToolsPort int
	Headless     bool
	App          bool
	WindowSize   string
	BrowserBin   string
	TempDir      bool
	Watch        bool
}
