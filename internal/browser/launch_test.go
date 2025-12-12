package browser

import (
	"slices"
	"testing"
)

func TestBuildLaunchArgs_NoBrittleAutomationFlags(t *testing.T) {
	args := buildLaunchArgs(LaunchOptions{
		DevToolsPort: 9222,
		StartURL:     "http://127.0.0.1:1234/",
		AppMode:      true,
		WindowSize:   "800,600",
	})

	if slices.Contains(args, "--disable-infobars") {
		t.Fatalf("args unexpectedly contain --disable-infobars: %v", args)
	}
	if slices.Contains(args, "--disable-blink-features=AutomationControlled") {
		t.Fatalf("args unexpectedly contain --disable-blink-features=AutomationControlled: %v", args)
	}
}

func TestBuildLaunchArgs_AppMode(t *testing.T) {
	args := buildLaunchArgs(LaunchOptions{
		DevToolsPort: 9222,
		StartURL:     "http://127.0.0.1:1234/",
		AppMode:      true,
	})

	if got, want := args[len(args)-1], "--app=http://127.0.0.1:1234/"; got != want {
		t.Fatalf("last arg mismatch: got %q want %q (args=%v)", got, want, args)
	}
}

func TestBuildLaunchArgs_HeadlessUsesURL(t *testing.T) {
	args := buildLaunchArgs(LaunchOptions{
		DevToolsPort: 9222,
		StartURL:     "http://127.0.0.1:1234/",
		AppMode:      true,
		Headless:     true,
	})

	if got := args[len(args)-1]; got != "http://127.0.0.1:1234/" {
		t.Fatalf("expected start URL as last arg in headless: got %q (args=%v)", got, args)
	}
	if !slices.Contains(args, "--headless=new") {
		t.Fatalf("expected --headless=new in args: %v", args)
	}
}

