package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/HITENDRAS940/hotreload/internal/builder"
	"github.com/HITENDRAS940/hotreload/internal/runner"
	"github.com/HITENDRAS940/hotreload/internal/ui"
	"github.com/HITENDRAS940/hotreload/internal/watcher"
)

type Orchestrator struct {
	w              *watcher.Watcher
	b              *builder.Builder
	r              *runner.Runner
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
			ui.Step("file change detected  →  rebuilding")
			o.triggerRebuild()

		case err := <-o.w.Errors():
			ui.Error("watcher error: " + err.Error())

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
		// stop happens silently — runner emits its own ui output
		o.r.Stop()
	}
	o.serverMutex.Unlock()

	o.buildMutex.Lock()
	if o.buildCancel != nil {
		// cancel silently
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
		// builder already printed ui.Fail — nothing more to do here
		return
	}

	o.serverMutex.Lock()
	defer o.serverMutex.Unlock()

	if o.r != nil && !o.r.IsRunning() {
		// runner emits its own started ui output
		o.crashMutex.Lock()
		o.crashTimes = make([]time.Time, 0)
		o.crashMutex.Unlock()

		err := o.r.Start()
		if err != nil {
			ui.Error("failed to start server: " + err.Error())
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
	ui.Error("server crashed unexpectedly")

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
	ui.Warn(fmt.Sprintf("crash #%d detected in last 10s", crashCount))

	if crashCount >= 3 {
		ui.Error("crash loop detected (3+ crashes in 10s) — backing off 5s")
		o.crashMutex.Unlock()

		select {
		case <-time.After(5 * time.Second):
			ui.Info("resuming after crash-loop backoff")
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
	ui.Warn("shutting down")

	o.serverMutex.Lock()
	if o.r != nil && o.r.IsRunning() {
		o.r.Stop()
	}
	o.serverMutex.Unlock()

	o.buildMutex.Lock()
	if o.buildCancel != nil {
		o.buildCancel()
	}
	o.buildMutex.Unlock()

	o.mainCancel()
}
