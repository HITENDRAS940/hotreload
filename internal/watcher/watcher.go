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
	fw           *fsnotify.Watcher
	root         string
	debounc      *DebouncedSignal
	events       chan struct{}
	errChan      chan error
	done         chan struct{}
	mu           sync.Mutex
	watched      map[string]bool
	extraIgnore  []string
	watchAll     bool   // true when no .hotreloadignore — watch every file
	buildOutDir  string // directory of the exec binary — always ignored
}

func NewWatcher(root, execPath string) (*Watcher, error) {
	// Resolve the build-output directory so we never watch the binary being
	// written — otherwise the rebuild itself triggers the next rebuild.
	buildOutDir := ""
	if execPath != "" {
		if abs, err := filepath.Abs(filepath.Dir(execPath)); err == nil {
			buildOutDir = abs
		}
	}
	extraIgnore := LoadIgnorePatterns(root)

	// nil means the file was not found and user chose to continue without it —
	// in that mode we watch every file with no filtering at all.
	watchAll := extraIgnore == nil

	if !watchAll && len(extraIgnore) > 0 {
		ui.Info(fmt.Sprintf("loaded .hotreloadignore  (%d patterns)", len(extraIgnore)))
		for _, p := range extraIgnore {
			ui.Exclude(p)
		}
	} else if !watchAll {
		ui.Info("empty .hotreloadignore — watching all files")
	} else {
		ui.Info("no .hotreloadignore — watching every file change")
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
		watchAll:    watchAll,
		buildOutDir: buildOutDir,
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

		// When watchAll, skip nothing — add every directory
		// Exception: always skip the build-output directory to prevent loops.
		if w.buildOutDir != "" {
			if abs, err := filepath.Abs(path); err == nil && abs == w.buildOutDir {
				return filepath.SkipDir
			}
		}
		if !w.watchAll && shouldIgnore(path, w.extraIgnore) {
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

			// Always skip build-output directory events to prevent rebuild loops.
			if w.buildOutDir != "" {
				if abs, err := filepath.Abs(event.Name); err == nil {
					if abs == w.buildOutDir || len(abs) > len(w.buildOutDir) && abs[:len(w.buildOutDir)+1] == w.buildOutDir+"/" {
						continue
					}
				}
			}

			// When watchAll, skip shouldIgnore entirely
			if !w.watchAll && shouldIgnore(event.Name, w.extraIgnore) {
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

			// When watchAll, every file event triggers a rebuild;
			// otherwise only .go files and extension-less files.
			if w.watchAll || isWatchable(event.Name) {
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
