// +build !windows

package terminator

import (
	"syscall"

	"github.com/shirou/gopsutil/v3/process"
)

// stop tries to gracefully terminate the process.
func stop(opts Options) error {
	// Close each child.
	if opts.Tree {
		list := []*process.Process{}
		err := GetTree(opts.Pid, &list, false)
		if err != nil {
			return err
		}
		for _, child := range list {
			_ = syscall.Kill(int(child.Pid), syscall.SIGINT)
			_ = syscall.Kill(int(child.Pid), syscall.SIGTERM)
		}
	}

	// Close the root process.
	_ = syscall.Kill(opts.Pid, syscall.SIGINT)
	_ = syscall.Kill(opts.Pid, syscall.SIGTERM)

	return nil
}

// writeAnswer writes an answer message to the console process if specified.
// TODO: Implement if needed.
func writeAnswer(pid int, answer string) error {
	return nil
}
