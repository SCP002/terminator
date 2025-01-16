//go:build !darwin && !windows

package terminator

import (
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/shirou/gopsutil/v4/process"
)

// GetTerm returns TTY device of the process with PID `pid`.
func GetTerm(pid int) (string, error) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Get terminal of process with PID %v: Create process object", pid))
	}
	term, err := proc.Terminal()
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Get terminal of process with PID %v", pid))
	}
	if term == "" {
		return "", errors.New(fmt.Sprintf("Get terminal of process with PID %v: Terminal not found", pid))
	}
	return "/dev" + term, nil
}
