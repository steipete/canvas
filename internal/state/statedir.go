package state

import (
	"os"
	"path/filepath"
)

// DefaultStateDir returns the directory used to store runtime session info.
// On macOS this is typically ~/Library/Application Support/canvas.
func DefaultStateDir() (string, error) {
	if override := os.Getenv("CANVAS_STATE_DIR"); override != "" {
		return override, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "canvas"), nil
}
