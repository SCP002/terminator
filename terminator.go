package terminator

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/shirou/gopsutil/v4/process"
)

// Kill is the same as KillWithContext with background context.
func Kill(pid int) error {
	return KillWithContext(context.Background(), pid)
}

// KillWithContext kills process with PID `pid` using context `ctx`.
func KillWithContext(ctx context.Context, pid int) error {
	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), fmt.Sprintf("Kill process with PID %v", pid))
	default:
		proc, err := process.NewProcess(int32(pid))
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Kill process with PID %v", pid))
		}
		return errors.Wrap(proc.Kill(), fmt.Sprintf("Kill process with PID %v", pid))
	}
}

// WaitForProcStop returns when process with PID `pid` is no longer running or `ctx` deadline exceedes.
func WaitForProcStop(ctx context.Context, pid int) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return
	}

	ticker := time.NewTicker(time.Millisecond * 100)
	for {
		select {
		case <-ticker.C:
			if running, _ := proc.IsRunning(); !running {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// FlatChildTree returns gopsutil Process instances of all descendants of a process with the specified PID `pid`.
//
// The first element is deepest descendant. The last one is a progenitor or closest child.
//
// If the `withRoot` argument is set to true, add root process to the end.
func FlatChildTree(pid int, withRoot bool) ([]*process.Process, error) {
	tree := []*process.Process{}
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return tree, errors.Wrap(err, "Get flat child process tree")
	}
	err = flatChildTree(proc, &tree, withRoot)
	if err != nil {
		return tree, errors.Wrap(err, "Get flat child process tree")
	}
	return tree, nil
}

// flatChildTree populates the `tree` argument with gopsutil Process instances of all descendants of the specified
// process `proc`.
//
// The first element in the tree is deepest descendant. The last one is a progenitor or closest child.
//
// If the `withRoot` argument is set to true, add the root process to the end.
func flatChildTree(proc *process.Process, tree *[]*process.Process, withRoot bool) error {
	children, err := proc.Children()
	if errors.Is(err, process.ErrorNoChildren) {
		return nil
	}
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Get flat process tree for PID %v", proc.Pid))
	}
	// Iterate for each child process in reverse order.
	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]
		// Call self to collect descendants.
		err := flatChildTree(child, tree, false)
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
