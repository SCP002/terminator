// +build windows

package terminator

import (
	"syscall"

	"github.com/gonutz/w32/v2"
)

// Stop tries to gracefully terminate process with the specified PID
// by sending CTRL_BREAK_EVENT (for console applications) and
// WM_CLOSE (for desktop applications) sequentially
func Stop(pid int) {
	sendCtrlBreak(pid)
	closeWindow(pid)
}

// sendCtrlBreak sends CTRL_BREAK_EVENT message to the console process with the specified PID
func sendCtrlBreak(pid int) {
	dll := syscall.MustLoadDLL("kernel32.dll")
	defer dll.Release()
	procedure := dll.MustFindProc("GenerateConsoleCtrlEvent")
	// If "r1" equals 0 then error is happened.
	// If caller is not an owner of the child process, then it always
	// will return 0 with error "The parameter is incorrect", even
	// if such PID exists, so ignoring return parameters.
	procedure.Call(syscall.CTRL_BREAK_EVENT, uintptr(pid))
}

// closeWindow sends WM_CLOSE message to the main window of the process with the specified PID
func closeWindow(pid int) {
	w32.EnumWindows(func(hwnd w32.HWND) bool {
		_, currentPid := w32.GetWindowThreadProcessId(hwnd)

		if int(currentPid) == pid && isMainWindow(hwnd) {
			// Ignoring result of method call, it always retuns 0
			w32.SendMessage(hwnd, w32.WM_CLOSE, 0, 0)
			// Stop enumerating
			return false
		}
		// Continue enumerating
		return true
	})
}

// isMainWindow returns if window with the specified handle is a main window
func isMainWindow(hwnd w32.HWND) bool {
	return w32.GetWindow(hwnd, w32.GW_OWNER) == 0 && w32.IsWindowVisible(hwnd)
}
