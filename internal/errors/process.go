package errors

import (
	"fmt"
)


// ProcDied indicates the process is already dead.
//
// Implements error interface.
type ProcDied struct {
	PID int
}

func (e ProcDied) Error() string {
	return fmt.Sprintf("The process with PID %v is already dead.", e.PID)
}

func NewProcDied(pid int) ProcDied {
	return ProcDied{PID: pid}
}
