//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

// setSysProcAttr puts the child process into its own process group so we can
// SIGTERM/SIGKILL the whole group at once on Unix.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup sends SIGTERM (force=false) or SIGKILL (force=true) to the
// entire process group identified by pgid.
func killProcessGroup(pgid int, force bool) {
	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}
	syscall.Kill(-pgid, sig) //nolint:errcheck
}
