//go:build windows

package cli

import "syscall"

// daemonSysProcAttr returns the SysProcAttr for daemonizing on Windows.
func daemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		// On Windows, CREATE_NEW_PROCESS_GROUP detaches from console
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
