package osx

import "testing"

func TestScriptForPID(t *testing.T) {
	got := scriptForPID(1234)
	want := `tell application "System Events" to set frontmost of (first process whose unix id is 1234) to true`
	if got != want {
		t.Fatalf("scriptForPID mismatch:\n got: %q\nwant: %q", got, want)
	}
}
