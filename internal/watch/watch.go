package watch

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Options struct {
	Debounce time.Duration
}

func WatchRecursive(ctx context.Context, root string, opts Options, onChange func()) error {
	if opts.Debounce <= 0 {
		opts.Debounce = 250 * time.Millisecond
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	if err := addRecursive(w, root); err != nil {
		return err
	}

	var (
		mu    sync.Mutex
		timer *time.Timer
	)
	trigger := func() {
		mu.Lock()
		defer mu.Unlock()
		if timer == nil {
			timer = time.AfterFunc(opts.Debounce, onChange)
			return
		}
		timer.Reset(opts.Debounce)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-w.Errors:
			if err != nil {
				return err
			}
		case ev := <-w.Events:
			// Track new directories so recursive watching keeps working.
			if ev.Op&fsnotify.Create != 0 {
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
					_ = addRecursive(w, ev.Name)
				}
			}
			trigger()
		}
	}
}

func addRecursive(w *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		return w.Add(p)
	})
}
