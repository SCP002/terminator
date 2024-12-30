package errors

import (
	"fmt"
)

// ProcDiedError indicates the process is already dead.
type ProcDiedError struct {
	PID int
}

// Error is used to implement error interface.
func (e ProcDiedError) Error() string {
	return fmt.Sprintf("The process with PID %v is already dead", e.PID)
}

// NewProcDiedError returns new ProcDiedError.
func NewProcDiedError(pid int) ProcDiedError {
	return ProcDiedError{PID: pid}
}
