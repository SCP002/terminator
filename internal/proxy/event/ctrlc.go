package event

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

// Send sends CTRL_C_EVENT to the console of the process with the specified PID.
func Send(pid int) {
	// Negative process identifiers are disallowed in Windows,
	// using it as a default value check.
	if pid == -1 {
		os.Exit(codes.WrongPid)
	}

	k32 := windows.MustLoadDLL("kernel32.dll")
	defer k32.Release()

	// Attach to the target process console (form a console process group).
	k32Proc := k32.MustFindProc("AttachConsole")
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

	// Enable Ctrl + C processing (just in case).
	k32Proc = k32.MustFindProc("SetConsoleCtrlHandler")
	r1, _, _ = k32Proc.Call(NULL, FALSE)
	if r1 == 0 {
		os.Exit(codes.EnableCtrlCFailed)
	}

	// Send Ctrl + C signal to the current console process group.
	k32Proc = k32.MustFindProc("GenerateConsoleCtrlEvent")
	// Not using CTRL_BREAK_EVENT (which can't be ignored by the process) or
	// else, if our parent process shares the same console with this process,
	// we will stop the parent and SetConsoleCtrlHandler can't prevent it.
	// Parameter is 0 (all processes attached to the current console) but not
	// pid, or else it will fail to send a signal to consoles separate from
	// the current.
	r1, _, _ = k32Proc.Call(windows.CTRL_C_EVENT, uintptr(0))
	if r1 == 0 {
		os.Exit(codes.SendCtrlCFailed)
	}

	// If this program runs properly, we should never reach this point, rather
	// exit with STATUS_CONTROL_C_EXIT exit code generated by the system itself
	// (instantly if binary was built with -ldflags -H=windowsgui).
	// Exiting with proper exit code manually as a fallback.
	os.Exit(wincodes.STATUS_CONTROL_C_EXIT)
}