// +build windows

package terminator

import (
	"errors"
	"fmt"
	"syscall"

	"github.com/gonutz/w32/v2"
)

// stop tries to gracefully terminate process with the specified PID
// by sending WM_CLOSE (to desktop applications) or
// CTRL_BREAK_EVENT (to console applications).
func stop(pid int) {
	if isDesktopApp(pid) {
		// Calling closeWindow() to console application cause a few blank lines to appear in output
		closeWindow(pid)
	} else {
		// Calling sendCtrlBreak() to desktop application casue close button become unresponsive
		sendCtrlBreak(pid)
	}
}

// isDesktopApp returns true if process with the specified PID is a desktop application
func isDesktopApp(pid int) bool {
	_, err := getWindow(pid)
	return err == nil
}

// sendCtrlBreak sends CTRL_BREAK_EVENT message to the console process with the specified PID
func sendCtrlBreak(pid int) {
	dll := syscall.MustLoadDLL("kernel32.dll")
	defer dll.Release()
	procedure := dll.MustFindProc("GenerateConsoleCtrlEvent")
	// If "r1" equals 0 then error is happened.
	// If caller is not an owner of the child process, then it Always
	// will return 0 with error "The parameter is incorrect", even
	// if such PID exists, so ignoring return parameters.
	procedure.Call(syscall.CTRL_BREAK_EVENT, uintptr(pid))
}

// closeWindow sends WM_CLOSE message to the main window of the process with the specified PID
func closeWindow(pid int) {
	wnd, err := getWindow(pid)
	if err == nil {
		// Ignoring result of method call, it Always retuns 0
		w32.SendMessage(wnd, w32.WM_CLOSE, 0, 0)
	}
	// Not returning Any error because if we can not rely on result of SendMessage,
	// error value of closeWindow becomes inconsistent
}

// getWindow returns main window handle of the process with the specified PID
func getWindow(pid int) (w32.HWND, error) {
	var wnd w32.HWND
	w32.EnumWindows(func(hwnd w32.HWND) bool {
		_, currentPid := w32.GetWindowThreadProcessId(hwnd)

		if int(currentPid) == pid && isMainWindow(hwnd) {
			wnd = hwnd
			// Stop enumerating
			return false
		}
		// Continue enumerating
		return true
	})
	if wnd != 0 {
		return wnd, nil
	} else {
		return wnd, errors.New("No window found for PID " + fmt.Sprint(pid))
	}
}

// isMainWindow returns true if window with the specified handle is a main window
func isMainWindow(hwnd w32.HWND) bool {
	return w32.GetWindow(hwnd, w32.GW_OWNER) == 0 // && w32.IsWindowVisible(hwnd)
}
