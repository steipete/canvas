//go:build linux

package browser

import (
	"errors"
	"os/exec"
)

func FindChromiumBinary() (string, error) {
	for _, name := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "brave-browser", "microsoft-edge", "chrome"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("no Chromium browser found on PATH (install Chromium or set --browser-bin)")
}
