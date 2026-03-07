package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/HITENDRAS940/hotreload/internal/builder"
	"github.com/HITENDRAS940/hotreload/internal/config"
	"github.com/HITENDRAS940/hotreload/internal/orchestrator"
	"github.com/HITENDRAS940/hotreload/internal/runner"
	"github.com/HITENDRAS940/hotreload/internal/watcher"
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

	var r *runner.Runner
	if cfg.ExecCmd != "" {
		r = runner.NewRunner(cfg.ExecCmd)
	}

	orch := orchestrator.New(w, b, r)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go orch.Run()

	sig := <-sigChan
	slog.Info("shutdown signal received", "signal", sig)
	orch.Shutdown()
}
