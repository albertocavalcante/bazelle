//go:build unix

package cli

import "syscall"

// daemonSysProcAttr returns the SysProcAttr for daemonizing on Unix.
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true, // Create new session (detach from terminal)
	}
}
