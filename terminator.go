package terminator

// Stop tries to gracefully terminate process with the specified PID
func Stop(pid int) {
	stop(pid)
}
