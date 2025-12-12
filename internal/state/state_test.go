package state

import (
	"os"
	"testing"
	"time"
)

func TestSaveLoadRemove(t *testing.T) {
	dir := t.TempDir()
	in := Session{
		PID:        123,
		StartedAt:  time.Unix(1, 2).UTC(),
		Dir:        "/tmp/canvas",
		HTTPAddr:   "127.0.0.1",
		HTTPPort:   9999,
		SocketPath: "/tmp/canvas.sock",
		Token:      "abc",
		Headless:   true,
		BrowserBin: "/bin/chrome",
	}
	if err := Save(dir, in); err != nil {
		t.Fatal(err)
	}

	out, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if out.PID != in.PID || out.HTTPPort != in.HTTPPort || out.Token != in.Token || out.Dir != in.Dir {
		t.Fatalf("loaded session mismatch: %#v", out)
	}

	if err := Remove(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(SessionPath(dir)); !os.IsNotExist(err) {
		t.Fatalf("expected session file removed; stat err=%v", err)
	}
}
