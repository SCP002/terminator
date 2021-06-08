// +build !windows

package terminator

import "syscall"

// stop tries to gracefully terminate the process by sending SIGTERM to it or to a process group if such exits and the
// Tree option is "true".
func stop(opts Options) error {
	if opts.Tree {
		pgid, err := syscall.Getpgid(opts.Pid)
		if err == nil {
			opts.Pid = -pgid
		}
	}
	err := syscall.Kill(opts.Pid, syscall.SIGTERM)
	if err != nil {
		return err
	}
}
