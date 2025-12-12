//go:build darwin

package browser

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

func FindChromiumBinary() (string, error) {
	// Prefer explicit PATH binaries if available.
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}

	// Common macOS application bundles.
	candidates := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
	}

	home, _ := os.UserHomeDir()
	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, "Applications/Google Chrome.app/Contents/MacOS/Google Chrome"),
			filepath.Join(home, "Applications/Chromium.app/Contents/MacOS/Chromium"),
			filepath.Join(home, "Applications/Brave Browser.app/Contents/MacOS/Brave Browser"),
			filepath.Join(home, "Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"),
		)
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", errors.New("no Chromium/Chrome browser found (set --browser-bin)")
}
