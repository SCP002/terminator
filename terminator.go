package terminator

import (
	"errors"
	"fmt"
)

// Stop tries to gracefully terminate process with the specified PID
func Stop(pid int) error {
	stop(pid)

	running, err := IsRunning(pid)
	if err != nil {
		return err
	}
	if running {
		return errors.New("Failed to stop the process with PID: " + fmt.Sprint(pid))
	}
	return nil
}

// IsRunning returns true if process with the specified PID exists
func IsRunning(pid int) (bool, error) {
	return isRunning(pid)
}
