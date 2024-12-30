//go:build !windows

package terminator

import (
	"os"
	"syscall"
	"unsafe"

	"github.com/cockroachdb/errors"
	"github.com/shirou/gopsutil/v4/process"
)

// stop tries to gracefully terminate the process and write a message `msg` to stdin if it's not empty.
func (procExt *ProcessExt) stop(msg string) {
	// Error checks after each attempt are done to be consistent with implementation for Windows.

	// Try SIGINT.
	err := procExt.SendSignal(syscall.SIGINT)
	if errors.Is(err, os.ErrProcessDone) {
		procExt.State = Died
		return
	}
	if err == nil {
		if running, err := procExt.IsRunning(); !running && err == nil {
			procExt.State = Stopped
			return
		}
	}
	// Try SIGTERM.
	if err := procExt.Terminate(); err == nil {
		if running, err := procExt.IsRunning(); !running && err == nil {
			procExt.State = Stopped
			return
		}
	}
	// Try to write a message.
	if msg != "" {
		if err := writeMessage(procExt.Process, msg); err == nil {
			if running, err := procExt.IsRunning(); !running && err == nil {
				procExt.State = Stopped
				return
			}
		}
	}
}

// writeMessage writes a `msg` message to the console process `proc`.
//
// Requires root privilegies (e.g. run as sudo).
func writeMessage(proc *process.Process, msg string) error {
	term, err := proc.Terminal()
	if err != nil {
		return errors.Wrap(err, "Write message to stdin of a process")
	}
	file, err := os.OpenFile("/dev"+term, os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, "Write message to stdin of a process")
	}
	defer file.Close()
	for _, char := range msg {
		_, _, err := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), syscall.TIOCSTI, uintptr(unsafe.Pointer(&char)))
		if err != 0 {
			return errors.Wrap(err, "Write message to stdin of a process")
		}
	}
	return nil
}
