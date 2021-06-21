// +build windows

package terminator

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/gonutz/w32/v2"
	"github.com/shirou/gopsutil/v3/process"
	"golang.org/x/sys/windows"

	proxyBin "github.com/SCP002/terminator/internal/proxy/bin"
	proxyVer "github.com/SCP002/terminator/internal/proxy/version"
	"github.com/SCP002/terminator/internal/wincodes"
)

// Dll files.

var (
	k32 = windows.NewLazyDLL("kernel32.dll")
)

// stop tries to gracefully terminate the process.
func stop(opts Options) error {
	// Close each child.
	if opts.Tree {
		tree := []*process.Process{}
		err := GetTree(opts.Pid, &tree, false)
		if err != nil {
			return err
		}
		for _, child := range tree {
			_ = sendCtrlC(int(child.Pid), false)
			_ = sendCtrlBreak(int(child.Pid), true)
			_ = closeWindow(int(child.Pid), false, true)
		}
	}

	// Close the root process. Calling sendCtrlC with "checkRun" set to "true" as stopping a child can close the parent.
	_ = sendCtrlC(opts.Pid, true)
	_ = sendCtrlBreak(opts.Pid, true)
	if opts.Answer != "" {
		_ = writeAnswer(opts.Pid, opts.Answer)
	}
	_ = closeWindow(opts.Pid, false, true)

	return nil
}

// sendCtrlC sends a CTRL_C_EVENT to the process.
//
// If "checkRun" is set to "true", return an error if the process is not running, for better performance.
//
// If target process was started with CREATE_NEW_PROCESS_GROUP creation flag and SysProcAttr.NoInheritHandles is set to
// "false", CTRL_C_EVENT will have no effect.
func sendCtrlC(pid int, checkRun bool) error {
	if checkRun {
		if running, _ := IsRunning(pid); !running {
			return errors.New("The process with PID " + fmt.Sprint(pid) + " does not exist.")
		}
	}
	return sendSig(pid, windows.CTRL_C_EVENT)
}

// sendCtrlBreak sends a CTRL_BREAK_EVENT to the process.
//
// If "checkRun" is set to "true", return an error if the process is not running, for better performance.
func sendCtrlBreak(pid int, checkRun bool) error {
	if checkRun {
		if running, _ := IsRunning(pid); !running {
			return errors.New("The process with PID " + fmt.Sprint(pid) + " does not exist.")
		}
	}
	// If target process shares the same console with this one, CTRL_BREAK_EVENT will stop this process and
	// SetConsoleCtrlHandler can't prevent it.
	if attached, _ := isAttachedToCaller(pid); !attached {
		return sendSig(pid, windows.CTRL_BREAK_EVENT)
	}
	return errors.New("The process with PID " + fmt.Sprint(pid) + " is attached to the current console.")
}

// sendSig sends a control signal to the console process.
//
// Return value (error) is "nil" only if proxy process successfully sent the signal, but not necessary means that the
// signal has been successfully received or processed.
//
// Inspired by https://stackoverflow.com/a/15281070, https://stackoverflow.com/a/2445728.
func sendSig(pid int, signal int) error {
	const NULL uintptr = 0
	const TRUE uintptr = 1
	const FALSE uintptr = 0

	if signal == windows.CTRL_C_EVENT {
		// Disable Ctrl + C processing. If we don't disable it here, then despite the fact we're enabling it in Another
		// Process later, if the target process is using the same console as the current process, this program will
		// terminate itself.
		k32Proc := k32.NewProc("SetConsoleCtrlHandler")
		r1, _, err := k32Proc.Call(NULL, TRUE)
		if r1 == 0 {
			return err
		}
	}

	proxyPath, err := getProxyPath()
	if err != nil {
		return err
	}
	// Start a kamikaze process to attach to the console of the target process and send a signal.
	// Such proxy process is required because:
	// If the target process has it's own, separate console, then to attach to that console we have to FreeConsole of
	// this process first, so, unless this process was started from cmd.exe, the current console will be destroyed by
	// the system, so, this process will lose it's original console (AllocConsole means lost previous output, probably
	// broken redirection etc).
	// See /internal/proxy/proxy.go for the source code.
	kamikaze := exec.Command(proxyPath, "-mode", "signal", "-pid", fmt.Sprint(pid), "-sig", fmt.Sprint(signal))
	attr := syscall.SysProcAttr{}
	attr.CreationFlags |= windows.DETACHED_PROCESS
	attr.NoInheritHandles = true
	kamikaze.SysProcAttr = &attr
	// We don't rely on error value from Run() as it will return error if exit code is not 0, but in our case, normal
	// exit code is STATUS_CONTROL_C_EXIT (even if CTRL_BREAK_EVENT was sent).
	_ = kamikaze.Run()

	if signal == windows.CTRL_C_EVENT {
		// Enable Ctrl + C processing back to make this program react on Ctrl + C accordingly again and prevent new
		// child processes from inheriting the disabled state.
		// Usually, similar algorithms wait for a few seconds before enabling Ctrl + C back again to prevent self kill
		// if SetConsoleCtrlHandler triggered before CTRL_C_EVENT is sent.
		// We omit such delay as the call to kamikaze.Run() stops the current goroutine until CTRL_C_EVENT is sent or
		// the process exited with unexpected exit code (CTRL_C_EVENT failed).
		k32Proc := k32.NewProc("SetConsoleCtrlHandler")
		r1, _, err := k32Proc.Call(NULL, FALSE)
		if r1 == 0 {
			return err
		}
	}

	// Return error if the kamikaze process failed to exit with STATUS_CONTROL_C_EXIT.
	exitCode := kamikaze.ProcessState.ExitCode()
	if exitCode != wincodes.STATUS_CONTROL_C_EXIT {
		return errors.New("The kamikaze process exited with unexpected exit code: " + fmt.Sprint(exitCode))
	}

	return nil
}

// writeAnswer writes an answer message to the console process if specified.
func writeAnswer(pid int, answer string) error {
	proxyPath, err := getProxyPath()
	if err != nil {
		return err
	}
	// Start a message sender process to attach to the console of the target process and write a message to it's input
	// using the -msg flag.
	// Such proxy process is required for the same reason as above.
	if answer != "" {
		msgSender := exec.Command(proxyPath, "-mode", "answer", "-pid", fmt.Sprint(pid), "-msg", answer)
		attr := syscall.SysProcAttr{}
		attr.CreationFlags |= windows.DETACHED_PROCESS
		attr.NoInheritHandles = true
		msgSender.SysProcAttr = &attr
		err = msgSender.Run()
		if err != nil {
			return errors.New("The message sender process exited with error: " + err.Error())
		}
	}
	return nil
}

// getProxyPath returns proxy executable path. Writes a binary to a temporary files folder if not exist already or if
// present version is different.
//
// Using embed binary instead of calling it directly by relative path to keep dependencies of a library user in a single
// file.
func getProxyPath() (string, error) {
	path := os.TempDir() + "\\terminator_proxy_" + proxyVer.Str + ".exe"

	// Binary is already exist.
	_, err := os.Stat(path)
	if err == nil {
		return path, nil
	}

	err = os.WriteFile(path, proxyBin.Bytes, 0755)
	if err != nil {
		return "", errors.New("Unable to write proxy binary: " + err.Error())
	}

	return path, nil
}

// closeWindow sends WM_CLOSE message to the main window of the process.
//
// If "allowOwnConsole" is set to "true", allow to close own console window of the process.
//
// If "checkRun" is set to "true", return an error if the process is not running, for better performance.
//
// Return value (error) is "nil" only if application successfully processes this message, but not necessary means that
// the window was actually closed.
func closeWindow(pid int, allowOwnConsole bool, checkRun bool) error {
	if checkRun {
		if running, _ := IsRunning(pid); !running {
			return errors.New("The process with PID " + fmt.Sprint(pid) + " does not exist.")
		}
	}
	wnd, err := getWindow(pid, allowOwnConsole)
	if err != nil {
		return err
	}
	r := w32.SendMessage(wnd, w32.WM_CLOSE, 0, 0)
	if r != 0 {
		return errors.New("Failed to close the window with PID " + fmt.Sprint(pid))
	}
	return nil
}

// getWindow returns main window handle of the process.
//
// If "allowOwnConsole" is set to "true", allow to return own console window of the process.
//
// Inspired by https://stackoverflow.com/a/21767578.
func getWindow(pid int, allowOwnConsole bool) (w32.HWND, error) {
	var wnd w32.HWND
	w32.EnumWindows(func(hwnd w32.HWND) bool {
		_, currentPid := w32.GetWindowThreadProcessId(hwnd)

		if int(currentPid) == pid && isMainWindow(hwnd) {
			wnd = hwnd
			// Stop enumerating.
			return false
		}
		// Continue enumerating.
		return true
	})
	if wnd != 0 {
		return wnd, nil
	} else {
		if allowOwnConsole {
			if attached, _ := isAttachedToCaller(pid); attached {
				return w32.GetConsoleWindow(), nil
			}
		}
		return wnd, errors.New("No window found for PID " + fmt.Sprint(pid))
	}
}

// isMainWindow returns true if a window with the specified handle is a main window.
func isMainWindow(hwnd w32.HWND) bool {
	return w32.GetWindow(hwnd, w32.GW_OWNER) == 0 && w32.IsWindowVisible(hwnd)
}

// isAttachedToCaller returns true if the given PID is attached to the current console.
func isAttachedToCaller(pid int) (bool, error) {
	pids, err := getConsolePids(1)
	if err != nil {
		return false, err
	}
	for _, currentPid := range pids {
		if currentPid == uint32(pid) {
			return true, nil
		}
	}
	return false, nil
}

// getConsolePids returns a slice of PID's attached to the current console.
//
// pidsLen parameter - the maximum number of PID's that can be stored in buffer.
// Must be > 0. Can be increased automatically (safe to pass 1).
//
// See https://docs.microsoft.com/en-us/windows/console/getconsoleprocesslist.
func getConsolePids(pidsLen int) ([]uint32, error) {
	k32Proc := k32.NewProc("GetConsoleProcessList")

	pids := make([]uint32, pidsLen)
	r1, _, err := k32Proc.Call(
		// Actually passing the whole slice. Must be [0] due the way syscall works.
		uintptr(unsafe.Pointer(&pids[0])),
		uintptr(pidsLen),
	)
	if r1 == 0 {
		return pids, err
	}
	if r1 <= uintptr(pidsLen) {
		// Success, return the slice.
		return pids, nil
	} else {
		// The initial buffer was too small. Call self again with the exact capacity.
		return getConsolePids(int(r1))
	}
}
