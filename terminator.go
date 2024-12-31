package terminator

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/shirou/gopsutil/v4/process"
)

// Options respresents options to stop a process.
type Options struct { // TODO: Pid to Message map?
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
	Message string
}

// State represents the process state.
type State string

const ( // TODO: Set initial state to "Unknown"
	Running State = "Running" // Is running.
	Stopped State = "Stopped" // Stopped gracefully.
	Killed  State = "Killed"  // Killed forcibly.
	Died    State = "Died"    // Terminated by unknown reason (killed by another process etc).
)

// ProcessExt represents gopsutil process extended with state info.
type ProcessExt struct {
	*process.Process
	State State
}

// newProcState returns the new default ProcState instance from gopsutil process `proc`.
func newProcessExt(proc *process.Process) ProcessExt {
	return ProcessExt{Process: proc, State: Running}
}

// killWithContext returns as soon as the process is stopped and kills when the context `ctx` Done() channel returns.
//
// `tick` is the interval at which the process status check will be performed.
func (procExt *ProcessExt) killWithContext(ctx context.Context, tick time.Duration) error {
	ticker := time.NewTicker(tick)
	for {
		select {
		case <-ticker.C:
			running, err := procExt.IsRunning()
			if !running && err == nil {
				procExt.State = Stopped
				return nil
			}
		case <-ctx.Done():
			err := procExt.Kill()
			if err == nil {
				procExt.State = Killed
			} else if errors.Is(err, os.ErrProcessDone) {
				procExt.State = Stopped
			} else {
				return errors.Wrap(err, fmt.Sprintf("Kill process with PID %v", procExt.Pid))
			}
			return nil
		}
	}
}

// StopResult represents mapping between PID's and current state for root and child processes.
type StopResult struct {
	Root     map[int32]State
	Children map[int32]State
}

// newStopResult returns new default StopResult instance with root process pid `rootPid`.
func newStopResult(rootPid int32) StopResult {
	return StopResult{
		Root:     map[int32]State{rootPid: Running},
		Children: map[int32]State{},
	}
}

// Stop tries to gracefully terminate a process with PID `pid` using options `opts`.
//
// Returns an error if process does not exist (if IgnoreAbsent option is set to false), if an internal error is happened
// or if failed to kill the root process or any child (if Tree option is set to true).
//
// --- On Windows it sequentially sends: ---
//
// A Ctrl + C signal (can be caught as a SIGINT).
//
// A Ctrl + Break signal (can be caught as a SIGINT).
//
// A WM_CLOSE (WM_QUIT for UWP apps) message (as if the user is closing the window, can be caught as a SIGTERM).
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
func Stop(pid int, opts Options) (StopResult, error) {
	rootProc, err := process.NewProcess(int32(pid))
	stopResult := newStopResult(rootProc.Pid)

	// Return if the process is not running.
	if err != nil {
		if opts.IgnoreAbsent {
			return stopResult, nil
		} else {
			return stopResult, errors.Wrap(err, fmt.Sprintf("Stop process with PID %v", rootProc.Pid))
		}
	}

	// Build the process tree.
	tree := []*process.Process{}
	if opts.Tree {
		err := GetTree(rootProc, &tree, false)
		if err != nil {
			return stopResult, errors.Wrap(err, fmt.Sprintf("Stop process with PID %v", rootProc.Pid))
		}
	}

	// Build ProcessExt list.
	childProcExtList := lo.Map(tree, func(proc *process.Process, _ int) *ProcessExt {
		p := newProcessExt(proc)
		return &p
	})

	// Try to stop child processes gracefully.
	for _, childProcExt := range childProcExtList {
		childProcExt.stop("")
	}

	// Try to stop the root process gracefully.
	rootProcExt := newProcessExt(rootProc)
	rootProcExt.stop(opts.Message)

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()
	var endErr error

	// Wait for child processes to stop in the allotted time and kill after timeout.
	var wg sync.WaitGroup
	// No need for `opts.Tree` check, `childProcExtList` is empty if `opts.Tree` is set to false.
	for _, child := range childProcExtList {
		if child.State == Running { // TODO: Use child.IsRunning()?
			wg.Add(1)
			go func() {
				err = child.killWithContext(ctx, opts.Tick)
				if err != nil {
					endErr = err
				}
				wg.Done()
			}()
		}
	}
	wg.Wait()

	// Wait for root process to stop in the allotted time and kill after timeout.
	if rootProcExt.State == Running {
		err = rootProcExt.killWithContext(ctx, opts.Tick)
		if err != nil {
			endErr = err
		}
	}

	// Build StopResult.
	stopResult.Root[rootProcExt.Pid] = rootProcExt.State
	stopResult.Children = lo.SliceToMap(childProcExtList, func(child *ProcessExt) (int32, State) {
		return child.Pid, child.State
	})

	return stopResult, endErr
}

// Kill terminates the process with PID `pid`.
//
// If `ignoreAbsent` is true, do not return error if process is not running (nothing to kill).
//
// If `withTree` is true, kill the specified process and any child processes which were started by it.
//
// Returns an error if the process does not exist (if `ignoreAbsent` is set to false), if internal error is happened
// or if failed to kill the root process or any child (if `withTree` is set to true).
func Kill(pid int, ignoreAbsent bool, withTree bool) error {
	proc, err := process.NewProcess(int32(pid))

	// Return if the process is not running.
	if err != nil {
		if ignoreAbsent {
			return nil
		} else {
			return errors.Wrap(err, fmt.Sprintf("Kill process with PID %v", proc.Pid))
		}
	}

	// Build the process tree.
	tree := []*process.Process{}
	if withTree {
		err := GetTree(proc, &tree, false)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Kill process with PID %v", proc.Pid))
		}
	}

	var endErr error

	// Try to kill child processes.
	for _, child := range tree {
		err = child.Kill()
		if !errors.Is(err, os.ErrProcessDone) && err != nil {
			endErr = errors.Wrap(err, fmt.Sprintf("Kill process with PID %v", child.Pid))
		}
	}

	// Try to kill root process.
	err = proc.Kill()
	if !errors.Is(err, os.ErrProcessDone) && err != nil {
		endErr = errors.Wrap(err, fmt.Sprintf("Kill process with PID %v", proc.Pid))
	}

	return endErr
}

// GetTree populates the `tree` argument with gopsutil Process instances of all descendants of the specified process
// `proc`.
//
// The first element in the tree is deepest descendant. The last one is a progenitor or closest child.
//
// If the `withRoot“ argument is set to true, include the root process.
func GetTree(proc *process.Process, tree *[]*process.Process, withRoot bool) error {
	children, err := proc.Children()
	if errors.Is(err, process.ErrorNoChildren) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Get process tree for PID %v", proc.Pid))
	}
	// Iterate for each child process in reverse order.
	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]
		// Call self to collect descendants.
		err := GetTree(child, tree, false)
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
