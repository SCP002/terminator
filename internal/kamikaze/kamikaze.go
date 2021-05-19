package main

import (
	"flag"
	"os"

	"golang.org/x/sys/windows"
)

/*
	Simple command line program which recieves PID of the process
	to send Ctrl + C signal to, and terminates together with the
	target process.

	Meant to be built with -ldflags -H=windowsgui build options to
	not to flash with the console during it's short life time and
	to not to call "FreeConsole" for nothing.

	Exit codes:
	0          - Application exited normally. NOT an expected value in our case.
	1          - Wrong PID.
	2          - AttachConsole failed.
	3          - SetConsoleCtrlHandler failed.
	4          - GenerateConsoleCtrlEvent failed.
	3221225786 - STATUS_CONTROL_C_EXIT, the application terminated as a result of a Ctrl + C, expected value.
*/

const (
	exitNormal int = iota
	exitWrongPid
	exitAttachFailed
	exitEnableCtrlCFailed
	exitSendCtrlCFailed
)

func main() {
	var pid int
	flag.IntVar(&pid, "pid", -1, "Process identifier of the console to attach to")
	flag.Parse()

	// Negative process identifiers are disallowed on Windows,
	// using it as a default value check.
	if pid == -1 {
		os.Exit(exitWrongPid)
	}

	dll := windows.MustLoadDLL("kernel32.dll")
	defer dll.Release()

	// Attach to the target process console (form a console process group).
	f := dll.MustFindProc("AttachConsole")
	r1, _, _ := f.Call(uintptr(pid))
	if r1 == 0 {
		os.Exit(exitAttachFailed)
	}

	// Enable Ctrl + C processing (just in case).
	f = dll.MustFindProc("SetConsoleCtrlHandler")
	const NULL uintptr = 0
	const FALSE uintptr = 0
	r1, _, _ = f.Call(NULL, FALSE)
	if r1 == 0 {
		os.Exit(exitEnableCtrlCFailed)
	}

	// Send Ctrl + C signal to the current console process group.
	// Not using CTRL_BREAK_EVENT (which can't be ignored by the process) or
	// else, if our parent process shares the same console with this process,
	// we will stop the parent and SetConsoleCtrlHandler can't protect it.
	f = dll.MustFindProc("GenerateConsoleCtrlEvent")
	r1, _, _ = f.Call(windows.CTRL_C_EVENT, uintptr(0)) // Not a typo, should be 0
	if r1 == 0 {
		os.Exit(exitSendCtrlCFailed)
	}

	// If this program runs properly, we should never reach this point, rather
	// exit with STATUS_CONTROL_C_EXIT (3221225786) exit code.
	os.Exit(exitNormal)
}
