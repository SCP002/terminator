//go:build windows && arm64

package wincodes

// Windows exit codes.

const (
	// The application has been terminated as a result of a Ctrl + C or Ctrl + Break. arm64 value.
	STATUS_CONTROL_C_EXIT int = 3221225786
)
