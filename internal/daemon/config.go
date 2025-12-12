package daemon

type Config struct {
	StateDir     string
	ServeDir     string
	HTTPPort     int
	DevToolsPort int
	Headless     bool
	BrowserBin   string
	TempDir      bool
	Watch        bool
}
