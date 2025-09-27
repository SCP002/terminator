//go:build !windows

package terminator

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/cockroachdb/errors"
	"github.com/shirou/gopsutil/v4/process"
	"golang.org/x/sys/unix"
)

// SendSignal is the same as SendSignalWithContext with background context.
func SendSignal(pid int, sig syscall.Signal) error {
	return SendSignalWithContext(context.Background(), pid, sig)
}

// SendSignalWithContext sends signal `sig` to the process with PID `pid` using context `ctx`.
func SendSignalWithContext(ctx context.Context, pid int, sig syscall.Signal) error {
	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), fmt.Sprintf("Send signal %v to the process with PID %v", sig, pid))
	default:
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return errors.Wrapf(err, "Send signal %v to the process with PID %v", sig, pid)
	}
	if err := proc.SendSignal(sig); err != nil {
		return errors.Wrapf(err, "Send signal %v to the process with PID %v", sig, pid)
	}
	return nil
}

// SendMessage is the same as SendMessageWithContext with background context.
func SendMessage(pid int, msg string) error {
	return SendMessageWithContext(context.Background(), pid, msg)
}

// SendMessageWithContext writes a `msg` message to the console process with PID `pid` using context `ctx`.
//
// `msg` must end with "\n" on Linux and with "\r" on macOS to be sent.
//
// Requires root privilegies (e.g. run as sudo).
func SendMessageWithContext(ctx context.Context, pid int, msg string) error {
	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), fmt.Sprintf("Write message to stdin of the process with PID %v", pid))
	default:
	}

	term, err := GetTerm(pid)
	if err != nil {
		return errors.Wrapf(err, "Write message to stdin of the process with PID %v", pid)
	}
	file, err := os.OpenFile(term, os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrapf(err, "Write message to stdin of the process with PID %v", pid)
	}
	defer file.Close()
	for _, char := range msg {
		if err = unix.IoctlSetPointerInt(int(file.Fd()), unix.TIOCSTI, int(char)); err != nil {
			return errors.Wrapf(err, "Write message to stdin of the process with PID %v", pid)
		}
	}
	return nil
}
