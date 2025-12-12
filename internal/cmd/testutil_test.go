package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func captureStdout(w *bytes.Buffer) (func() error, error) {
	old := os.Stdout
	r, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	os.Stdout = pw

	done := make(chan struct{})
	go func() {
		_, _ = w.ReadFrom(r)
		_ = r.Close()
		close(done)
	}()

	return func() error {
		_ = pw.Close()
		os.Stdout = old
		<-done
		return nil
	}, nil
}

func shortSocketPath(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("/tmp", "canvas-*.sock")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	_ = f.Close()
	_ = os.Remove(path)
	t.Cleanup(func() { _ = os.Remove(path) })
	return filepath.Clean(path)
}
