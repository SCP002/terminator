//go:build darwin

package terminator

import (
	"fmt"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"golang.org/x/sys/unix"
)

// GetTerm returns TTY device of the process with PID `pid`.
func GetTerm(pid int) (string, error) {
	kProc, err := unix.SysctlKinfoProc("kern.proc.pid", int(pid))
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Get terminal for PID %v: Get kernel process info", pid))
	}
	termMap, err := getTerminalMap()
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Get terminal for PID %v", pid))
	}
	term, ok := termMap[kProc.Eproc.Tdev]
	if !ok {
		return "", errors.New(fmt.Sprintf("Get terminal for PID %v: Terminal not found", pid))
	}
	return term, nil
}

// getTerminalMap returns mapping between 'sr_rdev' and TTY device.
func getTerminalMap() (map[int32]string, error) {
	out := make(map[int32]string)

	termFiles, err := filepath.Glob("/dev/tty*")
	if err != nil {
		return nil, errors.Wrap(err, "Get terminal map: List TTY devices")
	}
	for _, termFile := range termFiles {
		stat := unix.Stat_t{}
		if err := unix.Stat(termFile, &stat); err == nil {
			out[stat.Rdev] = termFile
		}
	}

	return out, nil
}
