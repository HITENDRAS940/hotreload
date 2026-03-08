package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/HITENDRAS940/hotreload/internal/ui"
)

type Watcher struct {
	fw          *fsnotify.Watcher
	root        string
	debounc     *DebouncedSignal
	events      chan struct{}
	errChan     chan error
	done        chan struct{}
	mu          sync.Mutex
	watched     map[string]bool
	extraIgnore []string
}

func NewWatcher(root string) (*Watcher, error) {
	extraIgnore := LoadIgnorePatterns(root)

	if len(extraIgnore) > 0 {
		ui.Info(fmt.Sprintf("loaded .hotreloadignore  (%d patterns)", len(extraIgnore)))
		for _, p := range extraIgnore {
			ui.Exclude(p)
		}
	} else {
		ui.Info("no .hotreloadignore — watching all files")
	}

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("cannot create fsnotify watcher: %w", err)
	}

	w := &Watcher{
		fw:          fw,
		root:        root,
		debounc:     NewDebouncedSignal(150 * time.Millisecond),
		events:      make(chan struct{}),
		errChan:     make(chan error, 10),
		done:        make(chan struct{}),
		watched:     make(map[string]bool),
		extraIgnore: extraIgnore,
	}

	if err := w.walkAndWatch(root); err != nil {
		fw.Close()
		return nil, err
	}

	ui.Watching(len(w.watched))

	go w.eventLoop()

	return w, nil
}

func (w *Watcher) walkAndWatch(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			ui.Warn(fmt.Sprintf("error walking %s: %v", path, err))
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		if shouldIgnore(path, w.extraIgnore) {
			return filepath.SkipDir
		}

		if err := w.fw.Add(path); err != nil {
			ui.Warn(fmt.Sprintf("could not watch %s: %v", path, err))
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

			if shouldIgnore(event.Name, w.extraIgnore) {
				continue
			}

			if event.Op&fsnotify.Create != 0 {
				info, err := os.Stat(event.Name)
				if err == nil && info.IsDir() {
					ui.Info("new directory — adding to watch: " + event.Name)
					w.fw.Add(event.Name)
					w.mu.Lock()
					w.watched[event.Name] = true
					w.mu.Unlock()
				}
			}

			if isWatchable(event.Name) {
				w.debounc.Trigger()
			}

		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			select {
			case w.errChan <- err:
			default:
				ui.Error("watcher error (buffer full): " + err.Error())
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
