package terminator

import (
	"errors"
	"fmt"

	"github.com/shirou/gopsutil/v3/process"
)

// Stop tries to gracefully terminate process with the specified PID
//
// If ignoreAbsent set to true, then do not return error if process
// is not running (nothing to stop)
func Stop(pid int, ignoreAbsent bool) error {
	running, err := IsRunning(pid)
	if err != nil {
		return err
	}
	if !running {
		if ignoreAbsent {
			return nil
		} else {
			return errors.New("Process with PID " + fmt.Sprint(pid) + " does not exist")
		}
	}

	// stop does not have any return values so relying on IsRunning for error checks
	stop(pid)

	running, err = IsRunning(pid)
	if err != nil {
		return err
	}
	if running {
		return errors.New("Failed to stop process with PID " + fmt.Sprint(pid))
	}
	return nil
}

// IsRunning returns true if process with the specified PID exists
func IsRunning(pid int) (bool, error) {
	return process.PidExists(int32(pid))
}
