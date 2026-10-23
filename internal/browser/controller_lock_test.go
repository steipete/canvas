package browser

import (
	"context"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func TestCloseReturnsWhileNavigateIsStuck(t *testing.T) {
	tabCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	started := make(chan struct{})
	run := func(ctx context.Context, actions ...chromedp.Action) error {
		if _, hasDeadline := ctx.Deadline(); hasDeadline {
			return nil
		}
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}

	controller := &Controller{tabCtx: tabCtx, cancelAll: cancel, runCDP: run}
	navDone := make(chan struct{})
	go func() {
		_, _, _ = controller.Navigate(context.Background(), "http://example.test")
		close(navDone)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("navigate did not start")
	}

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- controller.Close()
	}()
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close blocked behind Navigate")
	}

	select {
	case <-navDone:
	case <-time.After(time.Second):
		t.Fatal("Navigate stayed running after Close")
	}
}

func TestNavigateStopsWhenTheRequestEnds(t *testing.T) {
	tabCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	started := make(chan struct{})
	run := func(ctx context.Context, actions ...chromedp.Action) error {
		close(started)
		<-ctx.Done()
		return ctx.Err()
	}

	controller := &Controller{tabCtx: tabCtx, cancelAll: cancel, runCDP: run}
	reqCtx, reqCancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, _, err := controller.Navigate(reqCtx, "http://example.test")
		done <- err
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("navigate did not start")
	}
	reqCancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected cancellation")
		}
	case <-time.After(time.Second):
		t.Fatal("Navigate ignored the request context")
	}
}

func TestCanceledRequestDoesNotRunBrowserAction(t *testing.T) {
	tabCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	controller := &Controller{tabCtx: tabCtx, cancelAll: cancel, runCDP: func(context.Context, ...chromedp.Action) error {
		t.Fatal("canceled request reached browser")
		return nil
	}}
	ctx, requestCancel := context.WithCancel(context.Background())
	requestCancel()
	if _, _, err := controller.Navigate(ctx, "http://example.test"); err != context.Canceled {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestQueuedRequestCanBeCanceled(t *testing.T) {
	tabCtx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	started := make(chan struct{})
	controller := &Controller{tabCtx: tabCtx, cancelAll: cancel, runCDP: func(ctx context.Context, _ ...chromedp.Action) error {
		select {
		case <-started:
			t.Error("queued browser action ran concurrently")
		default:
			close(started)
		}
		<-ctx.Done()
		return ctx.Err()
	}}
	firstDone := make(chan struct{})
	go func() { _, _, _ = controller.Navigate(tabCtx, "http://example.test"); close(firstDone) }()
	<-started
	ctx, requestCancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, _, err := controller.Navigate(ctx, "http://example.test"); done <- err }()
	requestCancel()
	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("queued request: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled request remained queued")
	}
	cancel()
	<-firstDone
}
