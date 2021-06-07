package signal

import (
	"os"

	"golang.org/x/sys/windows"

	"github.com/SCP002/terminator/internal/proxy/codes"
	"github.com/SCP002/terminator/internal/wincodes"
)

// Windows constants.
const (
	NULL  uintptr = 0
	FALSE uintptr = 0
)

// Send sends a control signal to the console of the process.
func Send(pid int, sig int) {
	// Negative process identifiers are disallowed in Windows,
	// using it as a default value check.
	if pid == -1 {
		os.Exit(codes.WrongPid)
	}
	if sig != windows.CTRL_C_EVENT && sig != windows.CTRL_BREAK_EVENT {
		os.Exit(codes.WrongSig)
	}

	k32 := windows.NewLazyDLL("kernel32.dll")

	// Attach to the target process console (form a console process group).
	k32Proc := k32.NewProc("AttachConsole")
	r1, _, err := k32Proc.Call(uintptr(pid))
	if r1 == 0 {
		if err == windows.ERROR_ACCESS_DENIED {
			os.Exit(codes.CallerAlreadyAttached)
		}
		if err == windows.ERROR_INVALID_HANDLE {
			os.Exit(codes.TargetHaveNoConsole)
		}
		if err == windows.ERROR_INVALID_PARAMETER {
			os.Exit(codes.ProcessDoesNotExist)
		}
		os.Exit(codes.AttachFailed)
	}

	if sig == windows.CTRL_C_EVENT {
		// Enable Ctrl + C processing (in case if the current console
		// had it disabled).
		k32Proc = k32.NewProc("SetConsoleCtrlHandler")
		r1, _, _ = k32Proc.Call(NULL, FALSE)
		if r1 == 0 {
			os.Exit(codes.EnableCtrlCFailed)
		}
	}

	// Send the control signal to the current console process group.
	k32Proc = k32.NewProc("GenerateConsoleCtrlEvent")
	// Parameter is 0 (all processes attached to the current console) but not
	// "pid", or else it will fail to send the signal to a consoles separate
	// from the current one.
	r1, _, _ = k32Proc.Call(uintptr(sig), uintptr(0))
	if r1 == 0 {
		os.Exit(codes.SendSigFailed)
	}

	// If this program runs properly, it should never reach this point, rather
	// exit with STATUS_CONTROL_C_EXIT exit code generated by the system itself
	// (instantly if binary was built with -ldflags -H=windowsgui).
	// Exiting with a proper exit code manually as a fallback.
	os.Exit(wincodes.STATUS_CONTROL_C_EXIT)
}