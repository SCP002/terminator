//go:build windows && 386

package wincodes

// Windows exit codes.

const (
	// The application has been terminated as a result of a Ctrl + C or Ctrl + Break. x32 value.
	STATUS_CONTROL_C_EXIT int = -1073741510
)
