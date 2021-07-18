// +build !windows

package terminator

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/shirou/gopsutil/v3/process"
)

// stop tries to gracefully terminate the process.
func (ps *ProcState) stop(answer string) {
	// Error checks after each attempt are done to be consistent with implementation for Windows.

	// Try SIGINT.
	err := ps.SendSignal(syscall.SIGINT)
	if err == os.ErrProcessDone {
		ps.State = Died
		return
	}
	if err == nil {
		if running, err := ps.IsRunning(); !running && err == nil {
			ps.State = Stopped
			return
		}
	}
	// Try SIGTERM.
	if err := ps.Terminate(); err == nil {
		if running, err := ps.IsRunning(); !running && err == nil {
			ps.State = Stopped
			return
		}
	}
	// Try to write an answer.
	if answer != "" {
		if err := writeAnswer(*ps.Process, answer); err == nil {
			if running, err := ps.IsRunning(); !running && err == nil {
				ps.State = Stopped
				return
			}
		}
	}
}

// writeAnswer writes an answer message to the console process.
//
// Requires root privilegies (e.g. run as sudo).
func writeAnswer(proc process.Process, answer string) error {
	term, err := proc.Terminal()
	if err != nil {
		return err
	}
	f, err := os.OpenFile("/dev"+term, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, c := range answer {
		_, _, err := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCSTI, uintptr(unsafe.Pointer(&c)))
		if err != 0 {
			return err
		}
	}
	return nil
}
