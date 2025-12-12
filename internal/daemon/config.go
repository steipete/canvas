package daemon

type Config struct {
	StateDir   string
	ServeDir   string
	HTTPPort   int
	Headless   bool
	BrowserBin string
	TempDir    bool
	Watch      bool
}
