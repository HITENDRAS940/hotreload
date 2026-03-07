package watcher

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	fw      *fsnotify.Watcher
	root    string
	debounc *DebouncedSignal
	events  chan struct{}
	errChan chan error
	done    chan struct{}
	mu      sync.Mutex
	watched map[string]bool
	logger  *slog.Logger
}

func NewWatcher(root string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("cannot create fsnotify watcher: %w", err)
	}

	w := &Watcher{
		fw:      fw,
		root:    root,
		debounc: NewDebouncedSignal(150 * time.Millisecond),
		events:  make(chan struct{}),
		errChan: make(chan error, 10),
		done:    make(chan struct{}),
		watched: make(map[string]bool),
		logger:  slog.Default(),
	}

	if err := w.walkAndWatch(root); err != nil {
		fw.Close()
		return nil, err
	}

	w.logger.Info("watcher initialized",
		"root", root,
		"watchedDirs", len(w.watched),
	)

	go w.eventLoop()

	return w, nil
}

func (w *Watcher) walkAndWatch(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			w.logger.Warn("error walking directory", "path", path, "error", err)
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		if shouldIgnore(path) {
			return filepath.SkipDir
		}

		if err := w.fw.Add(path); err != nil {
			w.logger.Warn("could not add directory to watcher", "path", path, "error", err)
			return nil
		}

		w.mu.Lock()
		w.watched[path] = true
		w.mu.Unlock()

		return nil
	})
}

func (w *Watcher) eventLoop() {
	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}

			if shouldIgnore(event.Name) {
				continue
			}

			if event.Op&fsnotify.Create != 0 {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					w.logger.Info("new directory detected, adding to watcher", "path", event.Name)
					w.fw.Add(event.Name)
					w.mu.Lock()
					w.watched[event.Name] = true
					w.mu.Unlock()
				}
			}

			if isWatchable(event.Name) {
				w.logger.Debug("file change detected", "path", event.Name, "op", event.Op)
				w.debounc.Trigger()
			}

		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			select {
			case w.errChan <- err:
			default:
				w.logger.Error("watcher error (buffer full)", "error", err)
			}
		}
	}
}

func (w *Watcher) Events() <-chan struct{} {
	return w.debounc.Out()
}

func (w *Watcher) Errors() <-chan error {
	return w.errChan
}

func (w *Watcher) Close() error {
	close(w.done)
	w.debounc.Close()
	return w.fw.Close()
}
