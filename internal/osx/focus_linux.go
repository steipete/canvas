//go:build linux

package osx

import (
	"fmt"
	"os"
	"os/exec"
)

func FocusPID(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid browser PID %d", pid)
	}
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") == "" {
		return fmt.Errorf("window focus on Linux requires Hyprland")
	}
	output, err := exec.Command("hyprctl", "dispatch", fmt.Sprintf(`hl.dsp.focus({ window = "pid:%d" })`, pid)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("focus browser through Hyprland: %w: %s", err, output)
	}
	return nil
}
