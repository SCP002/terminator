//go:build windows

package terminator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/cockroachdb/errors"
	"github.com/gonutz/w32/v2"
	"github.com/samber/lo"
	"golang.org/x/sys/windows"

	proxyBin "github.com/SCP002/terminator/internal/proxy/bin"
	"github.com/SCP002/terminator/internal/proxy/exitcodes"
	proxyVer "github.com/SCP002/terminator/internal/proxy/version"
	"github.com/SCP002/terminator/internal/wincodes"
)

var (
	kernel32 = windows.NewLazyDLL("kernel32.dll")
)

// ErrProcDied indicates the process is already dead.
type ErrProcDied struct {
	PID int
}

// Error is used to implement error interface.
func (e ErrProcDied) Error() string {
	return fmt.Sprintf("The process with PID %v is already dead", e.PID)
}

// newErrProcDied returns new ErrProcDied with PID `pid`.
func newErrProcDied(pid int) ErrProcDied {
	return ErrProcDied{PID: pid}
}

// ErrBadExitCode indicates that process exited with unexpected exit code.
type ErrBadExitCode struct {
	Code     int
	ProcName string
}

// Error is used to implement error interface.
func (e ErrBadExitCode) Error() string {
	return fmt.Sprintf("%v process exited with unexpected exit code %v", e.ProcName, e.Code)
}

// newErrBadExitCode returns new ErrBadExitCode with exit code `code` and process name `procName`.
func newErrBadExitCode(code int, procName string) ErrBadExitCode {
	return ErrBadExitCode{Code: code, ProcName: procName}
}

// SendCtrlC sends a CTRL_C_EVENT to the process with PID `pid`.
//
// If target process was started with CREATE_NEW_PROCESS_GROUP creation flag and SysProcAttr.NoInheritHandles is set to
// false, CTRL_C_EVENT will have no effect.
//
// Can be caught as SIGINT.
func SendCtrlC(pid int) error {
	return errors.Wrap(sendSig(pid, windows.CTRL_C_EVENT), "Failed to send CTRL_C_EVENT")
}

// SendCtrlBreak sends a CTRL_BREAK_EVENT to the process with PID `pid`.
//
// Can be caught as SIGINT.
func SendCtrlBreak(pid int) error {
	// If target process shares the same console with this one, CTRL_BREAK_EVENT will stop this process and
	// SetConsoleCtrlHandler can't prevent it.
	attached, err := isAttachedToCaller(pid)
	if err != nil {
		return errors.Wrap(err, "Failed to send CTRL_BREAK_EVENT")
	}
	if attached {
		msg := "Failed to send CTRL_BREAK_EVENT: The process with PID %v is attached to current console"
		return errors.New(fmt.Sprintf(msg, pid))
	}
	return errors.Wrap(sendSig(pid, windows.CTRL_BREAK_EVENT), "Failed to send CTRL_BREAK_EVENT")
}

// SendMessage writes a `msg` message to the console process with PID `pid`.
//
// `msg` must end with "\r\n" to be sent.
func SendMessage(pid int, msg string) error {
	proxyPath, err := getProxyPath()
	if err != nil {
		return err
	}
	// Start a message sender process to attach to the console of the target process and write a message to it's input
	// using the -msg flag.
	// Such proxy process is required for the same reason as for sendSig function.
	msgSender := exec.Command(proxyPath, "-mode", "message", "-pid", fmt.Sprint(pid), "-msg", msg)
	attr := syscall.SysProcAttr{}
	attr.CreationFlags |= windows.DETACHED_PROCESS
	attr.NoInheritHandles = true
	msgSender.SysProcAttr = &attr
	err = msgSender.Run()
	if msgSender.ProcessState.ExitCode() == exitcodes.ProcessDoesNotExist {
		return newErrProcDied(pid)
	}
	return errors.Wrap(err, "Failed to start message sender process")
}

// CloseWindow sends WM_CLOSE message to the window `wnd` or WM_QUIT message to UWP application window.
//
// If `wait` is set to true, wait for the window procedure to process the message. It will stop execution until user,
// for example, answer a confirmation dialogue box.
//
// Return value (error) is nil only if application successfully processes this message, but not necessarily means that
// the window was actually closed.
//
// Can be caught as SIGTERM.
func CloseWindow(wnd w32.HWND, wait bool) error {
	var ok bool
	message := lo.Ternary(IsUWPAppWindow(wnd), w32.WM_QUIT, w32.WM_CLOSE)
	if wait {
		ok = w32.SendMessage(wnd, uint32(message), 0, 0) == 0
	} else {
		ok = w32.PostMessage(wnd, uint32(message), 0, 0)
	}
	if !ok {
		return errors.New(fmt.Sprintf("Failed to send close message to window with window handle %v", wnd))
	}
	return nil
}

// GetMainWindow returns main window handle of the process with PID `pid`.
//
// If `allowOwnConsole` is set to true, allow to return own console window of the process.
func GetMainWindow(pid int, allowOwnConsole bool) (w32.HWND, error) {
	wnd, _ := lo.First(lo.Filter(GetWindows(pid), func(wnd w32.HWND, _ int) bool {
		return IsMainWindow(wnd)
	}))
	if wnd != 0 {
		return wnd, nil
	} else {
		if allowOwnConsole {
			if attached, _ := isAttachedToCaller(pid); attached {
				return w32.GetConsoleWindow(), nil
			}
		}
		return wnd, errors.New(fmt.Sprintf("Failed to get main window handle for PID %v", pid))
	}
}

// GetWindows returns all window handles of the process with PID `pid`.
//
// Inspired by https://stackoverflow.com/a/21767578.
func GetWindows(pid int) []w32.HWND {
	var windows []w32.HWND
	w32.EnumWindows(func(wnd w32.HWND) bool {
		_, currentPid := w32.GetWindowThreadProcessId(wnd)
		if int(currentPid) == pid {
			windows = append(windows, wnd)
		}
		// Continue enumerating.
		return true
	})
	return windows
}

// IsUWPAppWindow returns true if a window with the specified handle `wnd` is a window of Universal Windows Platform
// application.
func IsUWPAppWindow(wnd w32.HWND) bool {
	info, _ := w32.GetWindowInfo(wnd)
	return info.AtomWindowType == 49223
}

// IsMainWindow returns true if a window with the specified handle `wnd` is a main window.
//
// Inspired by https://stackoverflow.com/a/21767578.
func IsMainWindow(wnd w32.HWND) bool {
	return w32.GetWindow(wnd, w32.GW_OWNER) == 0 && w32.IsWindowVisible(wnd)
}

// sendSig sends a control signal `sig` to the console process with PID `pid`.
//
// Return value (error) is nil only if proxy process successfully sent the signal, but not necessarily means that the
// signal has been successfully received or processed.
// Among others, can return ErrProcDied and ErrBadExitCode errors defined in this package.
//
// Inspired by https://stackoverflow.com/a/15281070, https://stackoverflow.com/a/2445728.
func sendSig(pid int, sig int) error {
	const NULL uintptr = 0
	const TRUE uintptr = 1
	const FALSE uintptr = 0

	if sig == windows.CTRL_C_EVENT {
		// Disable Ctrl + C processing. If we don't disable it here, then despite the fact we're enabling it in Another
		// Process later, if the target process is using the same console as the current process, this program will
		// terminate itself.
		setConsoleCtrlHandler := kernel32.NewProc("SetConsoleCtrlHandler")
		r1, _, err := setConsoleCtrlHandler.Call(NULL, TRUE)
		if r1 == 0 {
			return errors.Wrap(err, fmt.Sprintf("Send signal %v to process with PID %v", sig, pid))
		}
	}

	proxyPath, err := getProxyPath()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Send signal %v to process with PID %v", sig, pid))
	}
	// Start a kamikaze process to attach to the console of the target process and send a signal.
	// Such proxy process is required because:
	// If the target process has it's own, separate console, then to attach to that console we have to FreeConsole of
	// this process first, so, unless this process was started from cmd.exe, the current console will be destroyed by
	// the system, so, this process will lose it's original console (AllocConsole means lost previous output, probably
	// broken redirection etc).
	// See /internal/proxy/proxy.go for the source code.
	kamikaze := exec.Command(proxyPath, "-mode", "signal", "-pid", fmt.Sprint(pid), "-sig", fmt.Sprint(sig))
	attr := syscall.SysProcAttr{}
	attr.CreationFlags |= windows.DETACHED_PROCESS
	attr.NoInheritHandles = true
	kamikaze.SysProcAttr = &attr
	// We don't rely on error value from Run() as it will return error if exit code is not 0, but in our case, normal
	// exit code is STATUS_CONTROL_C_EXIT (even if CTRL_BREAK_EVENT was sent).
	_ = kamikaze.Run()

	if sig == windows.CTRL_C_EVENT {
		// Enable Ctrl + C processing back to make this program react on Ctrl + C accordingly again and prevent new
		// child processes from inheriting the disabled state.
		// Usually, similar algorithms wait for a few seconds before enabling Ctrl + C back again to prevent self kill
		// if SetConsoleCtrlHandler triggered before CTRL_C_EVENT is sent.
		// We omit such delay as the call to kamikaze.Run() stops the current goroutine until CTRL_C_EVENT is sent or
		// the process exited with unexpected exit code (CTRL_C_EVENT failed).
		setConsoleCtrlHandler := kernel32.NewProc("SetConsoleCtrlHandler")
		r1, _, err := setConsoleCtrlHandler.Call(NULL, FALSE)
		if r1 == 0 {
			return err
		}
	}

	// Return error if the kamikaze process exited with ProcessDoesNotExist or not with STATUS_CONTROL_C_EXIT.
	exitCode := kamikaze.ProcessState.ExitCode()
	if exitCode == exitcodes.ProcessDoesNotExist {
		return newErrProcDied(pid)
	}
	if exitCode != wincodes.STATUS_CONTROL_C_EXIT {
		return newErrBadExitCode(exitCode, "Kamikaze")
	}

	return nil
}

// getProxyPath returns proxy executable path.
//
// It writes a binary to a temporary files folder if not exist already or if present version is different.
//
// Using embed binary instead of calling it directly by relative path to keep dependencies of a library user in a single
// file.
func getProxyPath() (string, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("terminator_proxy_%v.exe", proxyVer.Str))

	// Binary is already exist.
	_, err := os.Stat(path)
	if err == nil {
		return path, nil
	}

	err = os.WriteFile(path, proxyBin.Bytes, 0755)
	if err != nil {
		return "", errors.Wrap(err, "Write proxy binary")
	}

	return path, nil
}

// isAttachedToCaller returns true if `pid` is attached to the current console.
func isAttachedToCaller(pid int) (bool, error) {
	pids, err := getConsolePids(1)
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("Check if PID %v is attached to caller", pid))
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
// `pidsLen` is the maximum number of PID's that can be stored in buffer.
// Must be > 0. Can be increased automatically (safe to pass 1).
//
// See https://docs.microsoft.com/en-us/windows/console/getconsoleprocesslist.
func getConsolePids(pidsLen int) ([]uint32, error) {
	getConsoleProcessList := kernel32.NewProc("GetConsoleProcessList")

	pids := make([]uint32, pidsLen)
	r1, _, err := getConsoleProcessList.Call(
		// Actually passing the whole slice. Must be [0] due the way syscall works.
		uintptr(unsafe.Pointer(&pids[0])),
		uintptr(pidsLen),
	)
	if r1 == 0 {
		return pids, errors.Wrap(err, "Get PID's attached to current console")
	}
	if r1 <= uintptr(pidsLen) {
		// Success, return the slice.
		return pids, nil
	} else {
		// The initial buffer was too small. Call self again with the exact capacity.
		return getConsolePids(int(r1))
	}
}
