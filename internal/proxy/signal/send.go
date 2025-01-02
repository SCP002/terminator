//go:build windows

package signal

import (
	"os"

	"github.com/cockroachdb/errors"
	"golang.org/x/sys/windows"

	"github.com/SCP002/terminator/internal/proxy/exitcodes"
	"github.com/SCP002/terminator/internal/wincodes"
)

// Windows constants.
const (
	NULL  uintptr = 0
	FALSE uintptr = 0
)

// Send sends a control signal `sig` to the console of the process with PID `pid`.
func Send(pid int, sig int) {
	// Negative process identifiers are disallowed in Windows, using it as a default value check.
	if pid == -1 {
		os.Exit(exitcodes.WrongPid)
	}
	if sig != windows.CTRL_C_EVENT && sig != windows.CTRL_BREAK_EVENT {
		os.Exit(exitcodes.WrongSig)
	}

	kernel32 := windows.NewLazyDLL("kernel32.dll")

	// Attach to the target process console (form a console process group).
	attachConsole := kernel32.NewProc("AttachConsole")
	r1, _, err := attachConsole.Call(uintptr(pid))
	if r1 == 0 {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			os.Exit(exitcodes.CallerAlreadyAttached)
		}
		if errors.Is(err, windows.ERROR_INVALID_HANDLE) {
			os.Exit(exitcodes.TargetHaveNoConsole)
		}
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			os.Exit(exitcodes.ProcessDoesNotExist)
		}
		os.Exit(exitcodes.AttachFailed)
	}

	if sig == windows.CTRL_C_EVENT {
		// Enable Ctrl + C processing (in case if the current console had it disabled).
		setConsoleCtrlHandler := kernel32.NewProc("SetConsoleCtrlHandler")
		r1, _, _ = setConsoleCtrlHandler.Call(NULL, FALSE)
		if r1 == 0 {
			os.Exit(exitcodes.EnableCtrlCFailed)
		}
	}

	// Send the control signal to the current console process group.
	generateConsoleCtrlEvent := kernel32.NewProc("GenerateConsoleCtrlEvent")
	// Parameter is 0 (all processes attached to the current console) but not `pid`, or else it will fail to send the
	// signal to a consoles separate from the current one.
	r1, _, _ = generateConsoleCtrlEvent.Call(uintptr(sig), uintptr(0))
	if r1 == 0 {
		os.Exit(exitcodes.SendSigFailed)
	}

	// If this program runs properly, it should never reach this point, rather exit with STATUS_CONTROL_C_EXIT exit code
	// generated by the system itself (instantly if binary was built with -ldflags -H=windowsgui).
	// Exiting with a proper exit code manually as a fallback.
	os.Exit(wincodes.STATUS_CONTROL_C_EXIT)
}
