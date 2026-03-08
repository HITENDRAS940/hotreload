//go:build windows

package runner

import (
	"os/exec"
)

// setSysProcAttr is a no-op on Windows; process groups work differently.
func setSysProcAttr(cmd *exec.Cmd) {}

// killProcessGroup kills the process on Windows by terminating it directly.
// Windows does not support Unix-style process groups or SIGTERM/SIGKILL.
func killProcessGroup(pgid int, force bool) {
	// On Windows pgid is the PID. exec.Cmd.Process.Kill() is the portable way;
	// the Stop() caller already has a reference to r.cmd so this is handled
	// there via r.cmd.Process.Kill() — nothing extra needed here.
}
