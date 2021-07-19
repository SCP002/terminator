package terminator

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Options respresents options to stop a process.
type Options struct {
	// Do not return error if process is not running (nothing to stop)?
	IgnoreAbsent bool

	// Stop the specified process and any child processes which were started by it?
	Tree bool

	// Time allotted for the process to stop gracefully before it get killed.
	Timeout time.Duration

	// The interval at which the process status check will be performed to kill after timeout.
	Tick time.Duration

	// If not empty, is a message to send to input of the target console after a signal is sent.
	//
	// --- On Windows: ---
	//
	// It must end with "\r\n" to be sent.
	//
	// If StdIn is redirected, the prompt of a batch executable is skipped automatically (no need for an answer).
	//
	// If this program itself is launched from a batch file (e.g. run.cmd), the final prompt appears After this program
	// ends, thus answering to it is beyond the scope of this program.
	//
	// --- On POSIX: ---
	//
	// It must end with "\n" to be sent.
	Answer string
}

// State represents the process state.
type State string

const (
	Running State = "Running" // Initial state.
	Stopped State = "Stopped" // Stopped gracefully.
	Killed  State = "Killed"  // Killed by force.
	Died    State = "Died"    // Terminated by unknown reason (killed by another process etc).
)

// ProcState represents the gopsutil process with status after stop attempt.
type ProcState struct {
	*process.Process
	State State
}

// newProcState returns the new default ProcState instance.
func newProcState(proc *process.Process) ProcState {
	return ProcState{Process: proc, State: Running}
}

// killWithContext returns as soon as the process is stopped and kills when the context Done channel returns.
//
// tick argument is the interval at which the process status check will be performed.
func (ps *ProcState) killWithContext(ctx context.Context, tick time.Duration) error {
	ticker := time.NewTicker(tick)
	for {
		select {
		case <-ticker.C:
			running, err := ps.IsRunning()
			if !running && err == nil {
				ps.State = Stopped
				return nil
			}
		case <-ctx.Done():
			err := ps.Kill()
			if err == nil {
				ps.State = Killed
			} else if err == os.ErrProcessDone {
				ps.State = Stopped
			} else {
				return err
			}
			return nil
		}
	}
}

// StopResult represents a container for root and child processes after stop attempt.
type StopResult struct {
	Root     ProcState
	Children []ProcState
}

// newStopResult returns the new default StopResult instance.
func newStopResult(proc *process.Process) StopResult {
	return StopResult{Root: newProcState(proc), Children: []ProcState{}}
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
func Stop(pid int, opts Options) (StopResult, error) {
	proc, err := process.NewProcess(int32(pid))
	sr := newStopResult(proc)

	// Return if the process is not running.
	if err != nil {
		if opts.IgnoreAbsent {
			return sr, nil
		} else {
			return sr, err
		}
	}

	// Build the process tree.
	tree := []process.Process{}
	if opts.Tree {
		err := GetTree(*proc, &tree, false)
		if err != nil {
			return sr, err
		}
	}

	// Try to stop child processes gracefully.
	// TODO: One loop with kill?
	for i := range tree {
		child := &tree[i]
		ps := newProcState(child)
		ps.stop("")
		sr.Children = append(sr.Children, ps)
	}

	// Try to stop the root process gracefully.
	sr.Root.stop(opts.Answer)

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	var endErr error

	// Wait for child processes to stop in the allotted time and kill after timeout.
	var wg sync.WaitGroup
	for i := range sr.Children { // No need for opts.Tree check, sr.Children is empty if opts.Tree is "false".
		child := &sr.Children[i]
		if child.State == Running {
			wg.Add(1)
			go func() {
				err = child.killWithContext(ctx, opts.Tick)
				if endErr == nil {
					endErr = err
				}
				wg.Done()
			}()
		}
	}
	wg.Wait()

	// Wait for root process to stop in the allotted time and kill after timeout.
	if sr.Root.State == Running {
		err = sr.Root.killWithContext(ctx, opts.Tick)
		if endErr == nil {
			endErr = err
		}
	}

	return sr, endErr
}

// TODO: Kill()

// GetTree populates the "tree" argument with gopsutil Process instances of all descendants of the specified process.
//
// The first element in the tree is deepest descendant. The last one is a progenitor or closest child.
//
// If the "withRoot" argument is set to "true", include the root process.
func GetTree(proc process.Process, tree *[]process.Process, withRoot bool) error {
	children, err := proc.Children()
	if err == process.ErrorNoChildren {
		return nil
	}
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
