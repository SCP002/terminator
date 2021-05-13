// +build windows

package terminator

import (
	"errors"
	"fmt"

	"github.com/gonutz/w32/v2"
)

// stop tries to gracefully terminate process with the specified PID
// by sending WM_CLOSE (to desktop applications) or
// CTRL_C_EVENT (to console applications)
func stop(pid int) error {
	if hasWindow(pid) {
		// Calling closeWindow() to console application cause a few blank lines to appear in output
		return closeWindow(pid)
	} else {
		// Calling stopConsole() to desktop application casue close button become unresponsive
		return stopConsole(pid)
	}
}

// hasWindow returns true if process with the specified PID is has own window
// FIXME: False positive hasWindow for detached console apps. Document the case.
func hasWindow(pid int) bool {
	_, err := getWindow(pid)
	return err == nil
}

// stopConsole sends CTRL_C_EVENT message
// to the console process with the specified PID
// TODO: Try creating proxy killer process approach
// TODO: Fallback to https://github.com/HattoriLabUTSW/windows-kill?
func stopConsole(pid int) error {
	return nil
}

// TODO: Add GUI kill tree support (Enumerating child? WM_QUERYENDSESSION? AllocConsole? TaskKill?), also see:
// https://stanislavs.org/stopping-command-line-applications-programatically-with-ctrl-c-events-from-net/
// https://stackoverflow.com/questions/813086/can-i-send-a-ctrl-c-sigint-to-an-application-on-windows/15281070#15281070

// closeWindow sends WM_CLOSE message to the main window of the process with the specified PID
func closeWindow(pid int) error {
	wnd, err := getWindow(pid)
	if err != nil {
		return err
	}
	r := w32.SendMessage(wnd, w32.WM_CLOSE, 0, 0)
	// WM_CLOSE returns 0 if appication processes this message, not if
	// it did it's job successfully
	if r != 0 {
		return errors.New("Failed to close window with PID " + fmt.Sprint(pid))
	}
	return nil
}

// getWindow returns main window handle of the process with the specified PID
//
// Inspired by https://stackoverflow.com/a/21767578
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
	return w32.GetWindow(hwnd, w32.GW_OWNER) == 0 && w32.IsWindowVisible(hwnd)
}
