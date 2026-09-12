//go:build linux

package osx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFocusLinux(t *testing.T) {
	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "")
	if err := FocusPID(123); err == nil {
		t.Fatal("unsupported compositor accepted")
	}
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	t.Setenv("HYPRLAND_INSTANCE_SIGNATURE", "test")
	script := `#!/bin/sh
[ "$1" = dispatch ] && [ "$2" = focuswindow ] && [ "$3" = pid:123 ]
`
	if err := os.WriteFile(filepath.Join(dir, "hyprctl"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	if err := FocusPID(123); err != nil {
		t.Fatal(err)
	}
	if err := FocusPID(-1); err == nil {
		t.Fatal("negative PID accepted")
	}
}
