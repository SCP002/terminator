package message

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/SCP002/terminator/internal/proxy/codes"
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

// Send sends a message to the input of the target console.
func Send(pid int, msg string) {
	// Negative process identifiers are disallowed in Windows,
	// using it as a default value check.
	if pid == -1 {
		os.Exit(codes.WrongPid)
	}
	if msg == "" {
		os.Exit(codes.NoMessage)
	}

	k32 := windows.MustLoadDLL("kernel32.dll")
	defer k32.Release()

	// Attach to the target process console.
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

	// Regain standard IO handles after AttachConsole.
	initConsoleHandles()

	// Write message to the current console's input.
	inpRecList, err := strToInputRecords(msg)
	if err != nil {
		os.Exit(codes.ConvertMsgFailed)
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
		os.Exit(codes.WriteMsgFailed)
	}
}

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
		os.Exit(codes.GetStdInHandleFailed)
	}
	hOut, err := windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil {
		os.Exit(codes.GetStdOutHandleFailed)
	}
	hErr, err := windows.GetStdHandle(windows.STD_ERROR_HANDLE)
	if err != nil {
		os.Exit(codes.GetStdErrHandleFailed)
	}

	// Wrap handles in files. /dev/ prefix just to make it conventional.
	stdInF := os.NewFile(uintptr(hIn), "/dev/stdin")
	if stdInF == nil {
		os.Exit(codes.MakeStdInFileFailed)
	}
	stdOutF := os.NewFile(uintptr(hOut), "/dev/stdout")
	if stdOutF == nil {
		os.Exit(codes.MakeStdOutFileFailed)
	}
	stdErrF := os.NewFile(uintptr(hErr), "/dev/stderr")
	if stdErrF == nil {
		os.Exit(codes.MakeStdErrFileFailed)
	}

	// Set handles for standard input, output and error devices.
	err = windows.SetStdHandle(windows.STD_INPUT_HANDLE, windows.Handle(stdInF.Fd()))
	if err != nil {
		os.Exit(codes.SetStdInHandleFailed)
	}
	err = windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, windows.Handle(stdOutF.Fd()))
	if err != nil {
		os.Exit(codes.SetStdOutHandleFailed)
	}
	err = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(stdErrF.Fd()))
	if err != nil {
		os.Exit(codes.SetStdErrHandleFailed)
	}

	// Update golang standard IO file descriptors.
	os.Stdin = stdInF
	os.Stdout = stdOutF
	os.Stderr = stdErrF
}
