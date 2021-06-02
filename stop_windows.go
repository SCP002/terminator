// +build windows

package terminator

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/gonutz/w32/v2"
	"golang.org/x/sys/windows"

	proxyBin "github.com/SCP002/terminator/internal/proxy/bin"
	proxyVer "github.com/SCP002/terminator/internal/proxy/version"
	"github.com/SCP002/terminator/internal/wincodes"
)

// stop tries to gracefully terminate the process.
func stop(opts Options) error {
	// TODO: Add If logic between closeWindow() and sendCtrlC() (just a parameter?)
	// Calling closeWindow() to console application cause a few blank lines to appear in output.
	// Calling sendCtrlC() to desktop application casue close button become unresponsive.
	return sendCtrlC(opts)
}

// TODO: Kill own / tree subprocess by iterating child (gopsutil.NewProcess + p.Children / TaskKill)?
// closeWindow sends WM_CLOSE message to the main window of the process.
func closeWindow(pid int) error {
	wnd, err := getWindow(pid)
	if err != nil {
		return err
	}
	r := w32.SendMessage(wnd, w32.WM_CLOSE, 0, 0)
	// WM_CLOSE returns 0 if appication processes this message, NOT if
	// it did it's job successfully!
	if r != 0 {
		return errors.New("Failed to close the window with PID " + fmt.Sprint(pid))
	}
	return nil
}

// getWindow returns main window handle of the process.
//
// Fails to detect own console window because if child process uses console of
// it's parent, then child don't have it's own window.
//
// Inspired by https://stackoverflow.com/a/21767578
func getWindow(pid int) (w32.HWND, error) {
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
		return wnd, errors.New("No window found for PID " + fmt.Sprint(pid))
	}
}

// isMainWindow returns true if a window with the specified handle is a main window.
func isMainWindow(hwnd w32.HWND) bool {
	return w32.GetWindow(hwnd, w32.GW_OWNER) == 0 && w32.IsWindowVisible(hwnd)
}

// getProxyPath returns proxy executable path. Writes a binary to a temporary
// files folder if not exist already or if present version is different.
//
// Using embed binary instead of calling it directly by relative path to
// keep dependencies of a library user in a single file.
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

// FIXME: Run basic from bath throws "Stop failed with: exit status 3" (send msg to self).
// TODO: Try to workaround / add note about child kill.

// sendCtrlC sends CTRL_C_EVENT to the console the process
// and writes an answer message if specified.
//
// Inspired by https://stackoverflow.com/a/15281070
func sendCtrlC(opts Options) error {
	const NULL uintptr = 0
	const TRUE uintptr = 1
	const FALSE uintptr = 0

	k32 := windows.MustLoadDLL("kernel32.dll")
	defer k32.Release()

	// Disable Ctrl + C processing. If we don't disable it here, then
	// despite the fact we're enabling it in Another Process later, if the
	// target process is using the same console as the current process, our
	// program will terminate itself.
	k32Proc := k32.MustFindProc("SetConsoleCtrlHandler")
	r1, _, err := k32Proc.Call(NULL, TRUE)
	if r1 == 0 {
		return err
	}

	// Start a kamikaze process to attach to the console of the target process
	// and send CTRL_C_EVENT.
	// Such proxy process is required because:
	// If the target process has it's own, separate console, then to
	// attach to that console we have to FreeConsole of this process first, so,
	// unless this process was started from cmd.exe, the current console will be
	// destroyed by the system, so, this process will lose it's original console
	// (AllocConsole means no previous output, probably broken redirection etc).
	// See /internal/proxy/proxy.go for the source code.
	proxyPath, err := getProxyPath()
	if err != nil {
		return err
	}
	kamikaze := exec.Command(proxyPath, "-mode", "ctrlc", "-pid", fmt.Sprint(opts.Pid))
	attr := syscall.SysProcAttr{}
	attr.CreationFlags |= windows.DETACHED_PROCESS
	attr.NoInheritHandles = true
	kamikaze.SysProcAttr = &attr
	// We don't rely on error value from Run() as it will return error if exit code is not 0,
	// but in our case, normal exit code is STATUS_CONTROL_C_EXIT.
	_ = kamikaze.Run()

	// Enable Ctrl + C processing back to make this program react on Ctrl + C
	// accordingly again and prevent new child processes from inheriting the disabled state.
	// Usually, similar algorithms wait for a few seconds before enabling Ctrl + C back
	// again to prevent self kill if SetConsoleCtrlHandler triggered before CTRL_C_EVENT is
	// sent. We omit such delay as the call to kamikaze.Run() stops the current goroutine until
	// CTRL_C_EVENT is sent or the process exited with unexpected exit code (CTRL_C_EVENT failed).
	k32Proc = k32.MustFindProc("SetConsoleCtrlHandler")
	r1, _, err = k32Proc.Call(NULL, FALSE)
	if r1 == 0 {
		return err
	}

	// Return error if the kamikaze process failed to exit with STATUS_CONTROL_C_EXIT
	// (target was not terminated with Ctrl + C).
	exitCode := kamikaze.ProcessState.ExitCode()
	if exitCode != wincodes.STATUS_CONTROL_C_EXIT {
		return errors.New("The kamikaze process exited with unexpected exit code: " + fmt.Sprint(exitCode))
	}

	// Start a message sender process to attach to the console of the target
	// process and write a message to it's input using the -msg flag.
	// Such proxy process is required for the same reason as above.
	if opts.Answer != "" {
		msgSender := exec.Command(proxyPath, "-mode", "answer", "-pid", fmt.Sprint(opts.Pid), "-msg", opts.Answer)
		attr = syscall.SysProcAttr{}
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

// func runTaskKill(pid int) error {
// 	cmd := exec.Command("TaskKill", "/Pid", fmt.Sprint(pid), "/T")
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
// 	return cmd.Run()
// }

// func enumThreadWindows(threadId uintptr, callback func(window w32.HWND) bool) bool {
// 	u32 := syscall.NewLazyDLL("user32.dll")
// 	u32Proc := u32.NewProc("EnumThreadWindows")
// 	cb := syscall.NewCallback(func(w, _ uintptr) uintptr {
// 		if callback(w32.HWND(w)) {
// 			return 1
// 		}
// 		return 0
// 	})
// 	ret, _, _ := u32Proc.Call(threadId, cb, 0)
// 	return ret != 0
// }
