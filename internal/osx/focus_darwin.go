//go:build darwin

package osx

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
)

func FocusPID(pid int) error {
	if pid <= 0 {
		return errors.New("invalid pid")
	}
	script := scriptForPID(pid)
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript failed: %w: %s", err, string(out))
	}
	return nil
}

func scriptForPID(pid int) string {
	return `tell application "System Events" to set frontmost of (first process whose unix id is ` + strconv.Itoa(pid) + `) to true`
}
