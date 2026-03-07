package watcher

import (
	"context"
	"time"
)

type DebouncedSignal struct {
	out      chan struct{}
	timer    *time.Timer
	debounce time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewDebouncedSignal(debounce time.Duration) *DebouncedSignal {
	ctx, cancel := context.WithCancel(context.Background())
	return &DebouncedSignal{
		out:      make(chan struct{}),
		debounce: debounce,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (ds *DebouncedSignal) Trigger() {
	if ds.timer != nil {
		ds.timer.Stop()
	}

	ds.timer = time.AfterFunc(ds.debounce, func() {
		select {
		case ds.out <- struct{}{}:
		case <-ds.ctx.Done():
		}
	})
}

func (ds *DebouncedSignal) Out() chan struct{} {
	return ds.out
}

func (ds *DebouncedSignal) Close() {
	if ds.timer != nil {
		ds.timer.Stop()
	}
	ds.cancel()
	close(ds.out)
}
