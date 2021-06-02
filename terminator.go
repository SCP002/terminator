package terminator

import (
	"errors"
	"fmt"

	"github.com/shirou/gopsutil/v3/process"
)

// TODO: Timeouts, see:
// https://dev.to/hekonsek/using-context-in-go-timeout-hg7
// https://stackoverflow.com/questions/61042141/stopping-running-function-using-context-timeout-in-golang

// Options respresents options to stop a process.
type Options struct {
	Pid          int    // Identifier of the process to stop.
	IgnoreAbsent bool   // Do not return error if process is not running (nothing to stop)?
	Tree         bool   // Stop the specified process and any child processes which were started by it?
	Timeout      int    // Time in milliseconds allotted for the process to stop gracefully before it get killed.
	Answer       string // If not empty, is a message to send to input of the target console after CTRL_C_EVENT is sent.
}

// Stop tries to gracefully terminate the process.
func Stop(opts Options) error {
	running, err := IsRunning(opts.Pid)
	if err != nil {
		return err
	}
	if !running {
		if opts.IgnoreAbsent {
			return nil
		} else {
			return errors.New("The process with PID " + fmt.Sprint(opts.Pid) + " does not exist.")
		}
	}

	// Can't fully rely on stop() return error value, so using IsRunning() for confirmation.
	err = stop(opts)
	if err != nil {
		return err
	}

	running, err = IsRunning(opts.Pid)
	if err != nil {
		return err
	}
	if running {
		return errors.New("Failed to stop the process with PID " + fmt.Sprint(opts.Pid))
	}
	return nil
}

// IsRunning returns true if the process exists.
func IsRunning(pid int) (bool, error) {
	return process.PidExists(int32(pid))
}
