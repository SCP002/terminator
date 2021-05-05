// +build !windows

package terminator

import "syscall"

// stop tries to gracefully terminate process with the specified PID
// by sending SIGTERM to the process or to the process group if such
// exits
func stop(pid int) {
	pgid, err := syscall.Getpgid(pid)
	if err == nil {
		pid = -pgid
	}
	syscall.Kill(pid, syscall.SIGTERM)
}
