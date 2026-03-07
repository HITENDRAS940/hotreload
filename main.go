package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	// Set up signal handlers for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Main event loop
	for {
		select {
		case <-w.Events():
			slog.Info("rebuild triggered")

		case err := <-w.Errors():
			slog.Error("watcher error", "error", err)

		case sig := <-sigChan:
			slog.Info("shutdown signal received", "signal", sig)
			return
		}
	}
}
