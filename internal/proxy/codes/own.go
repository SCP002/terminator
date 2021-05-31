package codes

// Own exit codes. Not using iota for clarity.

const (
	// Shared.

	WrongMode int = 1 // Wrong mode. Either not specified or invalid.
	WrongPid  int = 2 // Wrong PID. Either not specified or -1.

	CallerAlreadyAttached int = 3 // Calling process is already attached to a console.
	TargetHaveNoConsole   int = 4 // Target process does not have a console.
	ProcessDoesNotExist   int = 5 // Target process does not exist.
	AttachFailed          int = 6 // AttachConsole failed for an unknown reason.

	// "event" package.

	EnableCtrlCFailed int = 7 // SetConsoleCtrlHandler failed.
	SendCtrlCFailed   int = 8 // GenerateConsoleCtrlEvent failed.

	// "message" package.

	NoMessage int = 9 // Empty or no message specified.

	GetStdInHandleFailed  int = 10 // Failed to retrieve standard input handler.
	GetStdOutHandleFailed int = 11 // Failed to retrieve standard output handler.
	GetStdErrHandleFailed int = 12 // Failed to retrieve standard error handler.

	MakeStdInFileFailed  int = 13 // Failed to create a new file for standard input.
	MakeStdOutFileFailed int = 14 // Failed to create a new file for standard output.
	MakeStdErrFileFailed int = 15 // Failed to create a new file for standard error.

	SetStdInHandleFailed  int = 16 // Failed to set standard input handler.
	SetStdOutHandleFailed int = 17 // Failed to set standard output handler.
	SetStdErrHandleFailed int = 18 // Failed to set standard error handler.

	ConvertMsgFailed int = 19 // Failed to convert string message to an array of inputRecord.
	WriteMsgFailed   int = 20 // Failed to write an array of inputRecord to the current console's input.
)
