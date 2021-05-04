// +build windows

package terminator

import (
	"fmt"
	"os"
	"syscall"

	"github.com/gonutz/w32/v2"
)

// Stop tries to gracefully terminate process with the specified PID
// by sending CTRL_BREAK_EVENT and WM_CLOSE sequentially
func Stop(pid int) {
	SendCtrlBreak(pid)
	CloseWindow(pid)
}

// SendCtrlBreak sends CTRL_BREAK_EVENT message to the console process with the specified PID
func SendCtrlBreak(pid int) {
	d := syscall.MustLoadDLL("kernel32.dll")
	defer d.Release()
	p := d.MustFindProc("GenerateConsoleCtrlEvent")
	r, _, err := p.Call(syscall.CTRL_BREAK_EVENT, uintptr(pid))
	if r == 0 {
		fmt.Fprintln(os.Stderr, "GenerateConsoleCtrlEvent failed:", err)
	}
}

// CloseWindow sends WM_CLOSE message to the main window of the process with the specified PID
func CloseWindow(pid int) {
	w32.EnumWindows(func(hwnd w32.HWND) bool {
		_, currentPid := w32.GetWindowThreadProcessId(hwnd)

		if int(currentPid) == pid && isMainWindow(hwnd) {
			// Ignoring result of method call, it returns 0 in any case
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
