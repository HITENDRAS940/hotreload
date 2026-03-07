package watcher

import (
	"context"
	"time"
)

// DebouncedSignal manages a 150ms debounce window to collapse rapid file changes.
// Multiple events within the window are collapsed into a single trigger.
type DebouncedSignal struct {
	out      chan struct{}      // Output channel for debounced signals
	timer    *time.Timer        // Active timer (if any)
	debounce time.Duration      // Debounce window duration
	ctx      context.Context    // Context for lifecycle
	cancel   context.CancelFunc // Cancel function for cleanup
}

// NewDebouncedSignal creates a new debounced signal with the given debounce duration.
// It spawns a goroutine that sends a signal on the returned channel when the debounce fires.
func NewDebouncedSignal(debounce time.Duration) *DebouncedSignal {
	ctx, cancel := context.WithCancel(context.Background())
	return &DebouncedSignal{
		out:      make(chan struct{}),
		debounce: debounce,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Trigger marks that a change occurred. If the debounce timer is not running,
// it is started. If it is running, it is reset. Either way, a signal will be
// sent on Out() after the debounce duration expires without further calls.
func (ds *DebouncedSignal) Trigger() {
	if ds.timer != nil {
		// Reset existing timer
		ds.timer.Stop()
	}

	// Start new timer that will fire after debounce duration
	ds.timer = time.AfterFunc(ds.debounce, func() {
		select {
		case ds.out <- struct{}{}:
		case <-ds.ctx.Done():
			// Shutdown requested
		}
	})
}

// Out returns the output channel for debounced signals.
// A signal is sent on this channel when debounce fires (150ms after last Trigger).
func (ds *DebouncedSignal) Out() chan struct{} {
	return ds.out
}

// Close stops the debouncer and cleans up resources.
// After Close(), no further signals will be sent.
func (ds *DebouncedSignal) Close() {
	if ds.timer != nil {
		ds.timer.Stop()
	}
	ds.cancel()
	close(ds.out)
}
