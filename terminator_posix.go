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

// SendSigTerm is the same as SendSigTermWithContext with background context.
func SendSigTerm(pid int) error {
	return SendSigTermWithContext(context.Background(), pid)
}

// SendSigTermWithContext sends SIGTERM signal to the process with PID `pid` using context `ctx`.
func SendSigTermWithContext(ctx context.Context, pid int) error {
	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), fmt.Sprintf("Send SIGTERM to the process with PID %v", pid))
	default:
		break
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Send SIGTERM to the process with PID %v", pid))
	}
	if err := proc.Terminate(); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Send SIGTERM to the process with PID %v", pid))
	}
	return nil
}

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
		break
	}

	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Send signal %v to the process with PID %v", sig, pid))
	}
	if err := proc.SendSignal(sig); err != nil {
		return errors.Wrap(err, fmt.Sprintf("Send signal %v to the process with PID %v", sig, pid))
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
		break
	}

	term, err := GetTerm(pid)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Write message to stdin of the process with PID %v", pid))
	}
	file, err := os.OpenFile(term, os.O_WRONLY, 0644)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Write message to stdin of the process with PID %v", pid))
	}
	defer file.Close()
	for _, char := range msg {
		if err = unix.IoctlSetPointerInt(int(file.Fd()), unix.TIOCSTI, int(char)); err != nil {
			return errors.Wrap(err, fmt.Sprintf("Write message to stdin of the process with PID %v", pid))
		}
	}
	return nil
}
