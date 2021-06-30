// +build !windows

package terminator

import (
	"os"
	"syscall"

	"github.com/shirou/gopsutil/v3/process"
)

// stop tries to gracefully terminate the process.
func stop(proc process.Process, tree []process.Process, answer string) StopResult {
	sr := newStopResult(&proc)
	// Close each child if given.
	for _, child := range tree {
		psp := PostStopProc{Proc: &child}

		// Try SIGINT.
		err := child.SendSignal(syscall.SIGINT)
		if err == os.ErrProcessDone {
			psp.State = Died
			sr.Children = append(sr.Children, psp)
			continue
		}
		if err == nil {
			if running, err := child.IsRunning(); !running && err == nil {
				psp.State = Stopped
				sr.Children = append(sr.Children, psp)
				continue
			}
		}
		// Try SIGTERM.
		if err := child.Terminate(); err == nil {
			if running, err := child.IsRunning(); !running && err == nil {
				psp.State = Stopped
				sr.Children = append(sr.Children, psp)
			}
		}
	}
	// Close the root process.
	// Try SIGINT.
	err := proc.SendSignal(syscall.SIGINT)
	if err == os.ErrProcessDone {
		sr.Root.State = Died
		return sr
	}
	if err == nil {
		if running, err := proc.IsRunning(); !running && err == nil {
			sr.Root.State = Stopped
			return sr
		}
	}
	// Try SIGTERM.
	if err := proc.Terminate(); err == nil {
		if running, err := proc.IsRunning(); !running && err == nil {
			sr.Root.State = Stopped
			return sr
		}
	}
	// Try to write an answer.
	if answer != "" {
		// TODO: Implement it.
	}
	return sr
}
