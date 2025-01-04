//go:build !windows

package terminator

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/cockroachdb/errors"
	"github.com/shirou/gopsutil/v4/process"
)

// SendSigTerm sends SIGTERM signal to a process with PID `pid`.
func SendSigTerm(pid int) error {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Send SIGTERM to a process with PID %v", pid))
	}
	err = proc.Terminate()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Send SIGTERM to a process with PID %v", pid))
	}
	return nil
}

// SendSignal sends signal `sig` to a process with PID `pid`.
func SendSignal(pid int, sig syscall.Signal) error {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Send signal %v to a process with PID %v", sig, pid))
	}
	err = proc.SendSignal(sig)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Send signal %v to a process with PID %v", sig, pid))
	}
	return nil
}

// SendMessage writes a `msg` message to console process with PID `pid`.
//
// `msg` must end with "\n" to be sent.
//
// Requires root privilegies (e.g. run as sudo).
func SendMessage(pid int, msg string) error {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Write message to stdin of a process with PID %v", pid))
	}
	term, err := proc.Terminal()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Write message to stdin of a process with PID %v", pid))
	}
	file, err := os.OpenFile("/dev"+term, os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Write message to stdin of a process with PID %v", pid))
	}
	defer file.Close()
	for _, char := range msg {
		_, _, err := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), syscall.TIOCSTI, uintptr(unsafe.Pointer(&char)))
		if err != 0 {
			return errors.Wrap(err, fmt.Sprintf("Write message to stdin of a process with PID %v, Errno %v", pid, err))
		}
	}
	return nil
}
