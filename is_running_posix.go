// +build !windows

package terminator

import (
	"os"
	"syscall"
)

// isRunning returns true if process with the specified PID exists
func isRunning(pid int) (bool, error) {
	// On Unix it always return process and no error, ignoring err
	proc, _ := os.FindProcess(pid)
	// If sig is 0, then no signal is sent, but error checking is still performed
	err := proc.Signal(syscall.Signal(0))
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}
