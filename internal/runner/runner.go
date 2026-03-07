package runner

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Runner struct {
	cmdString string
	pgid      int
	cmd       *exec.Cmd
	logger    *slog.Logger
	mu        sync.Mutex
	running   bool
}

func NewRunner(cmdString string) *Runner {
	return &Runner{
		cmdString: cmdString,
		pgid:      0,
		cmd:       nil,
		logger:    slog.Default(),
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
	r.logger.Info("server starting")

	parts := parseShellCommand(r.cmdString)
	if len(parts) == 0 {
		return fmt.Errorf("invalid exec command: empty after parsing")
	}

	cmd := exec.Command(parts[0], parts[1:]...)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	r.logger.Debug("executing server command", "command", r.cmdString)

	err := cmd.Start()
	if err != nil {
		r.logger.Error("failed to start server", "error", err)
		return fmt.Errorf("server start failed: %w", err)
	}

	r.mu.Lock()
	r.cmd = cmd
	r.pgid = cmd.Process.Pid
	r.running = true
	r.mu.Unlock()

	r.logger.Info("server started",
		"pid", cmd.Process.Pid,
		"duration", time.Since(startTime).String(),
	)

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
		r.logger.Error("server exited with error", "error", err)
		return
	}

	r.logger.Info("server exited")
}

func (r *Runner) Stop() error {
	r.mu.Lock()
	if !r.running || r.cmd == nil {
		r.mu.Unlock()
		return fmt.Errorf("server is not running")
	}
	pgid := r.pgid
	r.mu.Unlock()

	r.logger.Info("stopping server", "pgid", pgid)

	syscall.Kill(-pgid, syscall.SIGTERM)
	r.logger.Debug("sent SIGTERM to process group", "pgid", pgid)

	timeout := time.After(3 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			r.logger.Warn("SIGTERM timeout, sending SIGKILL", "pgid", pgid)
			syscall.Kill(-pgid, syscall.SIGKILL)
			r.logger.Info("sent SIGKILL to process group", "pgid", pgid)

			r.mu.Lock()
			r.running = false
			r.mu.Unlock()

			return nil

		case <-ticker.C:
			r.mu.Lock()
			if !r.running {
				r.mu.Unlock()
				r.logger.Info("server stopped gracefully")
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
