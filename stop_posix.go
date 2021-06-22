// +build !windows

package terminator

import (
	"syscall"

	"github.com/shirou/gopsutil/v3/process"
)

// stop tries to gracefully terminate the process.
func stop(proc process.Process, tree []process.Process, answer string) {
	// Close each child if given.
	for _, child := range tree {
		_ = syscall.Kill(int(child.Pid), syscall.SIGINT)
		_ = syscall.Kill(int(child.Pid), syscall.SIGTERM)
	}

	// Close the root process.
	_ = syscall.Kill(int(proc.Pid), syscall.SIGINT)
	_ = syscall.Kill(int(proc.Pid), syscall.SIGTERM)
}
