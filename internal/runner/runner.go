package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/HITENDRAS940/hotreload/internal/ui"
)

type Runner struct {
	cmdString string
	pgid      int
	cmd       *exec.Cmd
	mu        sync.Mutex
	running   bool
}

func NewRunner(cmdString string) *Runner {
	return &Runner{
		cmdString: cmdString,
		pgid:      0,
		cmd:       nil,
		running:   false,
	}
}

func (r *Runner) Start() error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	r.mu.Unlock()

	startTime := time.Now()
	ui.Step("server starting")

	parts := parseShellCommand(r.cmdString)
	if len(parts) == 0 {
		return fmt.Errorf("invalid exec command: empty after parsing")
	}

	cmd := exec.Command(parts[0], parts[1:]...)

	setSysProcAttr(cmd)

	sw := ui.ServerWriter()
	cmd.Stdout = sw
	cmd.Stderr = sw
	cmd.Stdin = os.Stdin

	err := cmd.Start()
	if err != nil {
		ui.Error("failed to start server: " + err.Error())
		return fmt.Errorf("server start failed: %w", err)
	}

	r.mu.Lock()
	r.cmd = cmd
	r.pgid = cmd.Process.Pid
	r.running = true
	r.mu.Unlock()

	ui.Done(fmt.Sprintf("server started  (pid: %d)", cmd.Process.Pid), time.Since(startTime).Round(time.Millisecond).String())

	go r.waitForExit()

	return nil
}

func (r *Runner) waitForExit() {
	if r.cmd == nil {
		return
	}

	err := r.cmd.Wait()

	r.mu.Lock()
	r.running = false
	r.mu.Unlock()

	if err != nil {
		ui.Fail("server exited with error", err.Error())
		return
	}

	ui.Info("server exited")
}

func (r *Runner) Stop() error {
	r.mu.Lock()
	if !r.running || r.cmd == nil {
		r.mu.Unlock()
		return fmt.Errorf("server is not running")
	}
	pgid := r.pgid
	cmd := r.cmd
	r.mu.Unlock()

	ui.Step(fmt.Sprintf("stopping server  (pgid: %d)", pgid))

	killProcessGroup(pgid, false)
	// Portable fallback (needed on Windows where killProcessGroup is a no-op).
	if cmd.Process != nil {
		cmd.Process.Signal(os.Interrupt) //nolint:errcheck
	}

	timeout := time.After(3 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			ui.Warn(fmt.Sprintf("SIGTERM timeout, sending SIGKILL  (pgid: %d)", pgid))
			killProcessGroup(pgid, true)
			ui.Warn("sent SIGKILL")

			r.mu.Lock()
			r.running = false
			r.mu.Unlock()

			return nil

		case <-ticker.C:
			r.mu.Lock()
			if !r.running {
				r.mu.Unlock()
				ui.Success("server stopped gracefully")
				return nil
			}
			r.mu.Unlock()
		}
	}
}

func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

func parseShellCommand(cmd string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range cmd {
		switch {
		case (ch == '"' || ch == '\'') && !inQuotes:
			inQuotes = true
			quoteChar = ch

		case ch == quoteChar && inQuotes:
			inQuotes = false
			quoteChar = 0

		case ch == ' ' && !inQuotes:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}

		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}
