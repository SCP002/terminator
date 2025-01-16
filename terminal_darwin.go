//go:build darwin

package terminator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"golang.org/x/sys/unix"
)

// GetTerm returns TTY of the process with PID `pid`.
func GetTerm(pid int) (string, error) {
	kProc, err := unix.SysctlKinfoProc("kern.proc.pid", int(pid))
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Get terminal for PID %v: Get kernel process info", pid))
	}
	ttyNr := uint64(kProc.Eproc.Tdev)

	termMap, err := getTerminalMap()
	if err != nil {
		return "", errors.Wrap(err, fmt.Sprintf("Get terminal for PID %v: Get terminal map", pid))
	}

	return termMap[ttyNr], nil
}

func getTerminalMap() (map[uint64]string, error) {
	out := make(map[uint64]string)
	var termFiles []string

	dev, err := os.Open("/dev")
	if err != nil {
		return nil, err
	}
	defer dev.Close()

	devNames, err := dev.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	for _, devName := range devNames {
		if strings.HasPrefix(devName, "/dev/tty") {
			termFiles = append(termFiles, "/dev/tty/"+devName)
		}
	}

	var ptsNames []string
	ptsDev, err := os.Open("/dev/pts")
	if err != nil {
		ptsNames, _ = filepath.Glob("/dev/ttyp*")
		if ptsNames == nil {
			return nil, err
		}
	}
	defer ptsDev.Close()

	if ptsNames == nil {
		defer ptsDev.Close()
		ptsNames, err = ptsDev.Readdirnames(-1)
		if err != nil {
			return nil, err
		}
		for _, ptsName := range ptsNames {
			termFiles = append(termFiles, "/dev/pts/"+ptsName)
		}
	} else {
		termFiles = ptsNames
	}

	for _, name := range termFiles {
		stat := unix.Stat_t{}
		if err = unix.Stat(name, &stat); err != nil {
			return nil, err
		}
		rdev := uint64(stat.Rdev)
		out[rdev] = strings.Replace(name, "/dev", "", -1)
	}

	return out, nil
}
