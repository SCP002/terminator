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
	// Identifier of the process to stop.
	Pid int

	// Is a console process?
	Console bool

	// Signal type to send if Console is set to "true". 0 = CTRL_C_EVENT, 1 = CTRL_BREAK_EVENT.
	//
	// If target process was started with CREATE_NEW_PROCESS_GROUP creation flag and
	// SysProcAttr.NoInheritHandles is set to "false", CTRL_C_EVENT will have no effect.
	//
	// If target process shares the same console with this one, CTRL_BREAK_EVENT
	// will stop this process and SetConsoleCtrlHandler can't prevent it.
	Signal int

	// Do not return error if process is not running (nothing to stop)?
	IgnoreAbsent bool

	// Stop the specified process and any child processes which were started by it?
	Tree bool

	// Time in milliseconds allotted for the process to stop gracefully before it get killed.
	Timeout int

	// If not empty, is a message to send to input of the target console after a signal is sent.
	//
	// It must end with a Windows newline sequence ("\r\n") to be sent.
	//
	// If StdIn is redirected, the prompt of a batch executable is skipped automatically (no need for an answer).
	//
	// If this program itself is launched from a batch file
	// (e.g. run.cmd), prompt appears After this program ends,
	// thus answering is beyond the scope of this program
	// (no sense in answering).
	Answer string
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
