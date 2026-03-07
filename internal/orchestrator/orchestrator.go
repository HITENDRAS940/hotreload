package orchestrator

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/HITENDRAS940/hotreload/internal/builder"
	"github.com/HITENDRAS940/hotreload/internal/runner"
	"github.com/HITENDRAS940/hotreload/internal/watcher"
)

type Orchestrator struct {
	w              *watcher.Watcher
	b              *builder.Builder
	r              *runner.Runner
	logger         *slog.Logger
	buildMutex     sync.Mutex
	serverMutex    sync.Mutex
	buildCancel    context.CancelFunc
	mainCtx        context.Context
	mainCancel     context.CancelFunc
	crashTimes     []time.Time
	crashMutex     sync.Mutex
	needsInitBuild bool
}

func New(w *watcher.Watcher, b *builder.Builder, r *runner.Runner) *Orchestrator {
	mainCtx, mainCancel := context.WithCancel(context.Background())

	return &Orchestrator{
		w:              w,
		b:              b,
		r:              r,
		logger:         slog.Default(),
		buildCancel:    nil,
		mainCtx:        mainCtx,
		mainCancel:     mainCancel,
		crashTimes:     make([]time.Time, 0),
		needsInitBuild: true,
	}
}

func (o *Orchestrator) Run() {
	o.triggerRebuild()

	serverDeadChan := make(chan struct{})
	if o.r != nil {
		go o.monitorServer(serverDeadChan)
	}

	for {
		select {
		case <-o.w.Events():
			o.logger.Info("file change detected")
			o.triggerRebuild()

		case err := <-o.w.Errors():
			o.logger.Error("watcher error", "error", err)

		case <-serverDeadChan:
			o.handleServerCrash()

		case <-o.mainCtx.Done():
			return
		}
	}
}

func (o *Orchestrator) triggerRebuild() {
	o.serverMutex.Lock()
	if o.r != nil && o.r.IsRunning() {
		o.logger.Info("stopping server before rebuild")
		o.r.Stop()
	}
	o.serverMutex.Unlock()

	o.buildMutex.Lock()
	if o.buildCancel != nil {
		o.logger.Info("cancelling previous build")
		o.buildCancel()
	}
	o.buildMutex.Unlock()

	buildCtx, cancel := context.WithCancel(o.mainCtx)
	o.buildMutex.Lock()
	o.buildCancel = cancel
	o.buildMutex.Unlock()

	go o.runBuild(buildCtx)
}

func (o *Orchestrator) runBuild(ctx context.Context) {
	err := o.b.Build(ctx)
	if err != nil {
		o.logger.Error("build failed", "error", err)
		return
	}

	o.serverMutex.Lock()
	defer o.serverMutex.Unlock()

	if o.r != nil && !o.r.IsRunning() {
		o.logger.Info("build succeeded, starting server")

		o.crashMutex.Lock()
		o.crashTimes = make([]time.Time, 0)
		o.crashMutex.Unlock()

		err := o.r.Start()
		if err != nil {
			o.logger.Error("failed to start server", "error", err)
		}
	}
}

func (o *Orchestrator) monitorServer(serverDeadChan chan struct{}) {
	for {
		select {
		case <-o.mainCtx.Done():
			return
		default:
		}

		if !o.r.IsRunning() {
			serverDeadChan <- struct{}{}
			return
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func (o *Orchestrator) handleServerCrash() {
	o.logger.Error("server crashed unexpectedly")

	o.crashMutex.Lock()
	now := time.Now()
	o.crashTimes = append(o.crashTimes, now)

	cutoff := now.Add(-10 * time.Second)
	validCrashes := make([]time.Time, 0)
	for _, t := range o.crashTimes {
		if t.After(cutoff) {
			validCrashes = append(validCrashes, t)
		}
	}
	o.crashTimes = validCrashes

	crashCount := len(o.crashTimes)
	o.logger.Warn("crash detected", "count_in_10s", crashCount)

	if crashCount >= 3 {
		o.logger.Error("crash loop detected (3+ crashes in 10s), waiting 5 seconds before resume")
		o.crashMutex.Unlock()

		select {
		case <-time.After(5 * time.Second):
			o.logger.Info("resuming after crash-loop backoff")
		case <-o.mainCtx.Done():
			return
		}

		o.crashMutex.Lock()
		o.crashTimes = make([]time.Time, 0)
		o.crashMutex.Unlock()
	} else {
		o.crashMutex.Unlock()
	}

	o.triggerRebuild()
}

func (o *Orchestrator) Shutdown() {
	o.logger.Info("orchestrator shutting down")

	o.serverMutex.Lock()
	if o.r != nil && o.r.IsRunning() {
		o.logger.Info("stopping server during shutdown")
		o.r.Stop()
	}
	o.serverMutex.Unlock()

	o.buildMutex.Lock()
	if o.buildCancel != nil {
		o.logger.Info("cancelling build during shutdown")
		o.buildCancel()
	}
	o.buildMutex.Unlock()

	o.mainCancel()
}
