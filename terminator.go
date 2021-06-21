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

	// Do not return error if process is not running (nothing to stop)?
	IgnoreAbsent bool

	// Stop the specified process and any child processes which were started by it?
	Tree bool

	// Time in milliseconds allotted for the process to stop gracefully before it get killed.
	Timeout int

	// If not empty, is a message to send to input of the target console after a signal is sent.
	//
	// On Windows it must end with "\r\n" to be sent.
	//
	// If StdIn is redirected, the prompt of a batch executable is skipped automatically (no need for an answer).
	//
	// If this program itself is launched from a batch file (e.g. run.cmd), the final prompt appears After this program
	// ends, thus answering to it is beyond the scope of this program.
	Answer string
}

// Stop tries to gracefully terminate the process.
//
// On Windows it sequentially sends:
//
// A Ctrl + C signal (can be caught as a SIGINT).
//
// A Ctrl + Break signal (can be caught as a SIGINT).
//
// A WM_CLOSE message (as if the user is closing the window, can be caught as a SIGTERM).
//
// On POSIX it sequentially sends:
//
// A SIGINT signal.
//
// A SIGTERM signal.
func Stop(opts Options) error {
	// TODO: Use start time + PID to check Unique PID.
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

// GetTree populates the "tree" argument with gopsutil Process instances of all descendants of the specified process.
//
// The first element in the tree is deepest descendant. The last one is a progenitor or closest child.
//
// If the "withRoot" argument is set to "true", include the root process.
func GetTree(pid int, tree *[]*process.Process, withRoot bool) error {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return err
	}
	children, err := proc.Children()
	if err != nil {
		return err
	}
	// Iterate for each child process in reverse order.
	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]
		// Call self to collect descendants.
		err := GetTree(int(child.Pid), tree, false)
		if err != nil {
			return err
		}
		// Add the child after it's descendants.
		*tree = append(*tree, child)
	}
	// Add the root process to the end.
	if withRoot {
		*tree = append(*tree, proc)
	}
	return nil
}
