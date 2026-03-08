package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/HITENDRAS940/hotreload/internal/builder"
	"github.com/HITENDRAS940/hotreload/internal/config"
	"github.com/HITENDRAS940/hotreload/internal/orchestrator"
	"github.com/HITENDRAS940/hotreload/internal/runner"
	"github.com/HITENDRAS940/hotreload/internal/ui"
	"github.com/HITENDRAS940/hotreload/internal/watcher"
)

func main() {
	ui.Banner()

	cfg := config.Parse()

	ui.Config("root", cfg.Root)
	ui.Config("build", cfg.BuildCmd)
	ui.Config("exec", cfg.ExecCmd)
	ui.Separator()

	w, err := watcher.NewWatcher(cfg.Root)
	if err != nil {
		ui.Fatal("failed to create watcher: " + err.Error())
	}
	defer w.Close()

	ui.Separator()

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
	ui.Warn("shutdown signal received: " + sig.String())
	orch.Shutdown()
}
