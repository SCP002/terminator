package terminator

import (
	"context"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/shirou/gopsutil/v4/process"
)

// Kill kills process with PID `pid`.
func Kill(pid int) error {
	return KillWithContext(context.Background(), pid)
}

// KillWithContext kills process with PID `pid` using context `ctx`.
func KillWithContext(ctx context.Context, pid int) error {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Kill process with PID %v", pid))
	}
	return errors.Wrap(proc.KillWithContext(ctx), fmt.Sprintf("Kill process with PID %v", pid))
}

// FlatTree returns gopsutil Process instances of all descendants of a process with the specified `pid`.
//
// The first element is deepest descendant. The last one is a progenitor or closest child.
//
// If the `withRoot` argument is set to true, add the root process to the end.
func FlatTree(pid int, withRoot bool) ([]*process.Process, error) {
	tree := []*process.Process{}
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return tree, errors.Wrap(err, "Get flat process tree")
	}
	err = flatTree(proc, &tree, withRoot)
	if err != nil {
		return tree, errors.Wrap(err, "Get flat process tree")
	}
	return tree, nil
}

// flatTree populates the `tree` argument with gopsutil Process instances of all descendants of the specified process
// `proc`.
//
// The first element in the tree is deepest descendant. The last one is a progenitor or closest child.
//
// If the `withRoot` argument is set to true, add the root process to the end.
func flatTree(proc *process.Process, tree *[]*process.Process, withRoot bool) error {
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
		err := flatTree(child, tree, false)
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
