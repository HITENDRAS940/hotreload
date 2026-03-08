//go:build windows

package runner

import (
	"fmt"
	"os/exec"
)

// setSysProcAttr is a no-op on Windows; process groups work differently.
func setSysProcAttr(cmd *exec.Cmd) {}

// killProcessGroup kills the entire process tree on Windows using taskkill.
// This is the only reliable way to terminate a process and all its children
// on Windows since Unix-style process groups do not exist.
func killProcessGroup(pgid int, force bool) {
	// /F = force, /T = include child tree, /PID = target by pid
	args := []string{"/F", "/T", "/PID", fmt.Sprintf("%d", pgid)}
	exec.Command("taskkill", args...).Run() //nolint:errcheck
}
