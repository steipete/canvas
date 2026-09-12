//go:build linux

package browser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindLinuxBrowser(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	if _, err := FindChromiumBinary(); err == nil {
		t.Fatal("expected missing browser error")
	}
	for _, name := range []string{"google-chrome-stable", "chromium"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
			t.Fatal(err)
		}
		got, err := FindChromiumBinary()
		if err != nil || got != path {
			t.Fatalf("got %q, %v; want %q", got, err, path)
		}
	}
}
