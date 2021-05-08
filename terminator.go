package terminator

import (
	"errors"
	"fmt"

	"github.com/shirou/gopsutil/v3/process"
)

// TODO: Timeouts, see:
// https://dev.to/hekonsek/using-context-in-go-timeout-hg7
// https://stackoverflow.com/questions/61042141/stopping-running-function-using-context-timeout-in-golang

// Stop tries to gracefully terminate process with the specified PID
//
// If ignoreAbsent set to true, then do not return error if process
// is not running (nothing to stop)
func Stop(pid int, ignoreAbsent bool) error {
	// TODO: Fails to detect console applications
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

	// TODO: Wait for window to close after message is sent
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
