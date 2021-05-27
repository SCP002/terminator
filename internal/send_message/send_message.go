package main

import (
	"flag"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

/*
	Simple command line program which recieves PID of the process
	and string message to write to standard input of that process.

	Message should end with a Windows newline sequence ("\r\n") to
	be sent.

	Meant to be built with -ldflags -H=windowsgui build options to
	not to flash with the console during it's short life time and
	to not to call "FreeConsole" for nothing.

	Exit codes:
	1  - Wrong PID. Either not specified or -1.
	2  - Empty or no message specified.
	3  - Calling process is already attached to a console.
	4  - Target process does not have a console.
	5  - Target process does not exist.
	6  - AttachConsole failed for an unknown reason.
	7  - Failed to retrieve standard input handler.
	8  - Failed to retrieve standard output handler.
	9  - Failed to retrieve standard error handler.
	10 - Failed to create a new file for standard input.
	11 - Failed to create a new file for standard output.
	12 - Failed to create a new file for standard error.
	13 - Failed to set standard input handler.
	14 - Failed to set standard output handler.
	15 - Failed to set standard error handler.
	16 - Failed to convert string message to an array of inputRecord.
	17 - Failed to write an array of inputRecord to the current
	     console's input.
*/

// Own exit codes.
const (
	exitWrongPid int = iota + 1
	exitNoMessage

	exitCallerAlreadyAttached
	exitTargetHaveNoConsole
	exitProcessDoesNotExist
	exitAttachFailed

	exitGetStdInHandleFailed
	exitGetStdOutHandleFailed
	exitGetStdErrHandleFailed

	exitMakeStdInFileFailed
	exitMakeStdOutFileFailed
	exitMakeStdErrFileFailed

	exitSetStdInHandleFailed
	exitSetStdOutHandleFailed
	exitSetStdErrHandleFailed

	exitConvertMsgFailed
	exitWriteMsgFailed
)

// Windows constants.
const (
	// https://docs.microsoft.com/en-us/windows/console/input-record-str#members.
	keyEvent uint16 = 0x0001
)

// Windows types.
//
// Inspired by https://github.com/Azure/go-ansiterm/blob/master/winterm/api.go.
type (
	// https://docs.microsoft.com/en-us/windows/console/input-record-str.
	inputRecord struct {
		eventType uint16
		keyEvent  keyEventRecord
	}

	// https://docs.microsoft.com/en-us/windows/console/key-event-record-str.
	keyEventRecord struct {
		keyDown         int32
		repeatCount     uint16
		virtualKeyCode  uint16
		virtualScanCode uint16
		unicodeChar     uint16
		controlKeyState uint32
	}
)

// strToInputRecords converts string into a slice of inputRecord, see
// https://docs.microsoft.com/en-us/windows/console/input-record-str.
func strToInputRecords(msg string) ([]inputRecord, error) {
	records := []inputRecord{}
	utf16chars, err := windows.UTF16FromString(msg)
	if err != nil {
		return records, err
	}
	for _, char := range utf16chars {
		record := inputRecord{
			eventType: keyEvent,
			keyEvent: keyEventRecord{
				// 1 = TRUE, the key is pressed.
				// Can omit key release events.
				keyDown:         1,
				repeatCount:     1,
				virtualKeyCode:  0,
				virtualScanCode: 0,
				unicodeChar:     char,
				controlKeyState: 0,
			},
		}
		records = append(records, record)
	}
	return records, nil
}

// initConsoleHandles initializes standard IO handles for the current console.
// Useful to call after AttachConsole or AllocConsole.
func initConsoleHandles() {
	// Retrieve standard handles.
	hIn, err := windows.GetStdHandle(windows.STD_INPUT_HANDLE)
	if err != nil {
		os.Exit(exitGetStdInHandleFailed)
	}
	hOut, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil {
		os.Exit(exitGetStdOutHandleFailed)
	}
	hErr, err := windows.GetStdHandle(windows.STD_ERROR_HANDLE)
	if err != nil {
		os.Exit(exitGetStdErrHandleFailed)
	}

	// Wrap handles in files. /dev/ prefix just to make it conventional.
	stdInF := os.NewFile(uintptr(hIn), "/dev/stdin")
	if stdInF == nil {
		os.Exit(exitMakeStdInFileFailed)
	}
	stdOutF := os.NewFile(uintptr(hOut), "/dev/stdout")
	if stdOutF == nil {
		os.Exit(exitMakeStdOutFileFailed)
	}
	stdErrF := os.NewFile(uintptr(hErr), "/dev/stderr")
	if stdErrF == nil {
		os.Exit(exitMakeStdErrFileFailed)
	}

	// Set handles for standard input, output and error devices.
	err = windows.SetStdHandle(windows.STD_INPUT_HANDLE, windows.Handle(stdInF.Fd()))
	if err != nil {
		os.Exit(exitSetStdInHandleFailed)
	}
	err = windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, windows.Handle(stdOutF.Fd()))
	if err != nil {
		os.Exit(exitSetStdOutHandleFailed)
	}
	err = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(stdErrF.Fd()))
	if err != nil {
		os.Exit(exitSetStdErrHandleFailed)
	}

	// Update golang standard IO file descriptors.
	os.Stdin = stdInF
	os.Stdout = stdOutF
	os.Stderr = stdErrF
}

func main() {
	// Using -h flag to display help won't work as we don't have a console and
	// attach to the foreign one. Usage messages are placeholders intended to
	// be read here.
	var pid int
	flag.IntVar(&pid, "pid", -1, "Process identifier of the console to attach to")
	// Message should end with a Windows newline sequence ("\r\n") to be sent.
	var msg string
	flag.StringVar(&msg, "msg", "", "Message to send to StdIn")
	flag.Parse()

	// Negative process identifiers are disallowed in Windows,
	// using it as a default value check.
	if pid == -1 {
		os.Exit(exitWrongPid)
	}
	if msg == "" {
		os.Exit(exitNoMessage)
	}

	k32 := windows.MustLoadDLL("kernel32.dll")
	defer k32.Release()

	// Attach to the target process console.
	k32Proc := k32.MustFindProc("AttachConsole")
	r1, _, err := k32Proc.Call(uintptr(pid))
	if r1 == 0 {
		if err == windows.ERROR_ACCESS_DENIED {
			os.Exit(exitCallerAlreadyAttached)
		}
		if err == windows.ERROR_INVALID_HANDLE {
			os.Exit(exitTargetHaveNoConsole)
		}
		if err == windows.ERROR_INVALID_PARAMETER {
			os.Exit(exitProcessDoesNotExist)
		}
		os.Exit(exitAttachFailed)
	}

	// Regain standard IO handles after AttachConsole.
	initConsoleHandles()

	// Write message to the current console's input.
	inpRecList, err := strToInputRecords(msg)
	if err != nil {
		os.Exit(exitConvertMsgFailed)
	}
	k32Proc = k32.MustFindProc("WriteConsoleInputW")
	var written uint32 = 0
	var toWrite uint32 = uint32(len(inpRecList))
	r1, _, _ = k32Proc.Call(
		os.Stdin.Fd(),
		// Actually passing the whole slice. Should be [0] due the way syscall works.
		uintptr(unsafe.Pointer(&inpRecList[0])),
		uintptr(toWrite),
		// A pointer the number of input records actually written. Not using it (placeholder).
		uintptr(unsafe.Pointer(&written)),
	)
	if r1 == 0 {
		os.Exit(exitWriteMsgFailed)
	}
}
