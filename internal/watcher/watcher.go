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

// Watcher recursively watches a directory tree for file changes,
// detects new directories at runtime, and fires debounced rebuild signals.
type Watcher struct {
	fw      *fsnotify.Watcher // Underlying fsnotify watcher
	root    string            // Root directory to watch
	debounc *DebouncedSignal  // 150ms debouncer
	events  chan struct{}     // Output channel for debounced events
	errChan chan error        // Error channel for async errors
	done    chan struct{}     // Signal to stop watching
	mu      sync.Mutex        // Protect watched map
	watched map[string]bool   // Track which paths are being watched
	logger  *slog.Logger      // Structured logger
}

// NewWatcher creates a new recursive directory watcher.
// It starts watching root and all subdirectories immediately,
// automatically detects new directories created at runtime,
// and fires debounced rebuild signals on file changes.
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

	// Walk root recursively and add all directories to the watcher
	if err := w.walkAndWatch(root); err != nil {
		fw.Close()
		return nil, err
	}

	w.logger.Info("watcher initialized",
		"root", root,
		"watchedDirs", len(w.watched),
	)

	// Start the event processing loop
	go w.eventLoop()

	return w, nil
}

// walkAndWatch recursively walks a directory tree and adds all valid directories to fsnotify.
func (w *Watcher) walkAndWatch(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			w.logger.Warn("error walking directory", "path", path, "error", err)
			return nil // Continue walking even if one dir fails
		}

		// Only process directories
		if !d.IsDir() {
			return nil
		}

		// Skip ignored directories
		if shouldIgnore(path) {
			return filepath.SkipDir
		}

		// Add this directory to fsnotify
		if err := w.fw.Add(path); err != nil {
			w.logger.Warn("could not add directory to watcher", "path", path, "error", err)
			return nil // Continue walking
		}

		w.mu.Lock()
		w.watched[path] = true
		w.mu.Unlock()

		return nil
	})
}

// eventLoop processes fsnotify events, detects new directories, and triggers debouncing.
func (w *Watcher) eventLoop() {
	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}

			// Skip ignored paths
			if shouldIgnore(event.Name) {
				continue
			}

			// Detect if a new directory was created
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

			// Only trigger rebuild for watchable files (.go files, directory events)
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

// Events returns the channel for debounced rebuild signals.
// A signal is sent on this channel within 150ms after file changes stop.
func (w *Watcher) Events() <-chan struct{} {
	return w.debounc.Out()
}

// Errors returns a channel for async watcher errors.
func (w *Watcher) Errors() <-chan error {
	return w.errChan
}

// Close stops the watcher and cleans up resources.
func (w *Watcher) Close() error {
	close(w.done)
	w.debounc.Close()
	return w.fw.Close()
}
