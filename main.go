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
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := config.Parse()

	slog.Info("hotreload started",
		"root", cfg.Root,
		"build", cfg.BuildCmd,
		"exec", cfg.ExecCmd,
	)

	w, err := watcher.NewWatcher(cfg.Root)
	if err != nil {
		slog.Error("failed to create watcher", "error", err)
		os.Exit(1)
	}
	defer w.Close()

	slog.Info("watcher initialized, watching for changes...")

	b := builder.NewBuilder(cfg.BuildCmd)

	var buildCancel context.CancelFunc
	var buildMutex sync.Mutex
	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-w.Events():
			slog.Info("rebuild triggered")

			buildMutex.Lock()
			if buildCancel != nil {
				slog.Info("cancelling previous build")
				buildCancel()
			}
			buildMutex.Unlock()

			buildCtx, cancel := context.WithCancel(mainCtx)
			buildMutex.Lock()
			buildCancel = cancel
			buildMutex.Unlock()

			go func(ctx context.Context) {
				err := b.Build(ctx)
				if err != nil {
					slog.Error("build failed, waiting for next change", "error", err)
				}
			}(buildCtx)

		case err := <-w.Errors():
			slog.Error("watcher error", "error", err)

		case sig := <-sigChan:
			slog.Info("shutdown signal received", "signal", sig)

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
