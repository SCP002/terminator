// +build windows

package terminator

import "os"

// isRunning returns true if process with the specified PID exists
func isRunning(pid int) (bool, error) {
	_, err := os.FindProcess(pid)
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}
