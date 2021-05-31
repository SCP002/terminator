// +build windows
// +build 386

package wincodes

// Windows exit codes.

const (
	STATUS_CONTROL_C_EXIT int = -1073741510 // The application terminated as a result of a Ctrl + C. x32 value.
)
