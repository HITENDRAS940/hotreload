package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"hotreload/internal/builder"
	"hotreload/internal/config"
	"hotreload/internal/watcher"
)

func main() {
	// Set up structured logging — human-readable, timestamped.
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Parse and validate CLI flags.
	cfg := config.Parse()

	slog.Info("hotreload started",
		"root", cfg.Root,
		"build", cfg.BuildCmd,
		"exec", cfg.ExecCmd,
	)

	// Phase 2: Create file watcher
	w, err := watcher.NewWatcher(cfg.Root)
	if err != nil {
		slog.Error("failed to create watcher", "error", err)
		os.Exit(1)
	}
	defer w.Close()

	slog.Info("watcher initialized, watching for changes...")

	// Phase 3: Create builder for executing build commands
	b := builder.NewBuilder(cfg.BuildCmd)

	// Context management for in-flight builds
	// When a new rebuild is triggered, the previous build is cancelled
	var buildCancel context.CancelFunc        // Cancel function for current build
	var buildMutex sync.Mutex                 // Protects buildCancel
	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	// Set up signal handlers for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Main event loop
	for {
		select {
		case <-w.Events():
			slog.Info("rebuild triggered")

			// Cancel any previous in-flight build
			buildMutex.Lock()
			if buildCancel != nil {
				slog.Info("cancelling previous build")
				buildCancel()
			}
			buildMutex.Unlock()

			// Create new context for this build
			buildCtx, cancel := context.WithCancel(mainCtx)
			buildMutex.Lock()
			buildCancel = cancel
			buildMutex.Unlock()

			// Run build in goroutine so watcher remains responsive
			go func(ctx context.Context) {
				err := b.Build(ctx)
				if err != nil {
					slog.Error("build failed, waiting for next change", "error", err)
					// Don't exit on build failure — just wait for next file change
				}
			}(buildCtx)

		case err := <-w.Errors():
			slog.Error("watcher error", "error", err)

		case sig := <-sigChan:
			slog.Info("shutdown signal received", "signal", sig)

			// Cancel any in-flight build
			buildMutex.Lock()
			if buildCancel != nil {
				slog.Info("cancelling build during shutdown")
				buildCancel()
			}
			buildMutex.Unlock()

			return
		}
	}
}
