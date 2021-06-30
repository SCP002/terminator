package terminator

import (
	"os"

	"github.com/shirou/gopsutil/v3/process"
)

// TODO: Kill on timeout + prevent blocking, see:
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

// State represents the process state.
type State string

const (
	Running State = "Running"
	Stopped State = "Stopped"
	Killed  State = "Killed"
	Died    State = "Died"
)

// PostStopProc represents the process with status after stop attempt.
type PostStopProc struct {
	Proc  *process.Process
	State State
	// TODO: Channel?
}

// StopResult represents a container for root and child processes after stop attempt.
type StopResult struct {
	Root     PostStopProc
	Children []PostStopProc
}

// Stop tries to gracefully terminate the process.
//
// --- On Windows it sequentially sends: ---
//
// A Ctrl + C signal (can be caught as a SIGINT).
//
// A Ctrl + Break signal (can be caught as a SIGINT).
//
// A WM_CLOSE message (as if the user is closing the window, can be caught as a SIGTERM).
//
// TerminateProcess syscall as a fallback.
//
// --- On POSIX it sequentially sends: ---
//
// A SIGINT signal.
//
// A SIGTERM signal.
//
// A SIGKILL signal as a fallback.
//
// Returns an error if the process does not exist (if IgnoreAbsent is "false"), if an internal error is happened, or if
// failed to kill the root process or any child (if Tree is "true").
func Stop(opts Options) (StopResult, error) {
	proc, err := process.NewProcess(int32(opts.Pid))
	sr := newStopResult(proc)

	if err != nil {
		if opts.IgnoreAbsent {
			return sr, nil
		} else {
			return sr, err
		}
	}

	tree := []process.Process{}
	if opts.Tree {
		err := GetTree(*proc, &tree, false)
		if err != nil {
			return sr, err
		}
	}

	sr = stop(*proc, tree, opts.Answer)
	var endErr error

	// No need for opts.Tree check, sr.Children is empty if opts.Tree is "false".
	for _, child := range sr.Children {
		if child.State == Running {
			err = child.Proc.Kill()
			if err == nil {
				child.State = Killed
			} else if err == os.ErrProcessDone {
				child.State = Died
			} else {
				endErr = err
			}
		}
	}
	if sr.Root.State == Running {
		err = sr.Root.Proc.Kill()
		if err == nil {
			sr.Root.State = Killed
		} else if err == os.ErrProcessDone {
			sr.Root.State = Died
		} else {
			endErr = err
		}
	}

	return sr, endErr
}

// GetTree populates the "tree" argument with gopsutil Process instances of all descendants of the specified process.
//
// The first element in the tree is deepest descendant. The last one is a progenitor or closest child.
//
// If the "withRoot" argument is set to "true", include the root process.
func GetTree(proc process.Process, tree *[]process.Process, withRoot bool) error {
	children, err := proc.Children()
	if err != nil {
		return err
	}
	// Iterate for each child process in reverse order.
	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]
		// Call self to collect descendants.
		err := GetTree(*child, tree, false)
		if err != nil {
			return err
		}
		// Add the child after it's descendants.
		*tree = append(*tree, *child)
	}
	// Add the root process to the end.
	if withRoot {
		*tree = append(*tree, proc)
	}
	return nil
}

// newStopResult returns the new default StopResult instance.
func newStopResult(proc *process.Process) StopResult {
	return StopResult{
		Root: PostStopProc{
			Proc:  proc,
			State: Running,
		},
		Children: []PostStopProc{},
	}
}
