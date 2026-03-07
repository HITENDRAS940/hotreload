package main

import (
	"log/slog"
	"os"

	"hotreload/internal/config"
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

	// Phase 2+ will wire watcher → builder → runner here.
	slog.Info("config validated — ready for watcher (Phase 2)")
}
