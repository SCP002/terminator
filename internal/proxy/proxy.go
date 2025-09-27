//go:build windows

package main

import (
	"flag"
	"os"

	"github.com/SCP002/terminator/internal/proxy/exitcodes"
	"github.com/SCP002/terminator/internal/proxy/message"
	"github.com/SCP002/terminator/internal/proxy/signal"
)

/*
	Work modes:
	"signal": Sends a signal to a process and terminates with it.
	"message": Writes a message to the standard input of a process.

	Meant to be built with -ldflags "-H=windowsgui" build options to not to flash with the console during it's
	short life time and to not to call "FreeConsole" for nothing.
*/

func main() {
	// Using -h flag to display help won't work as we don't have a console and attach to the foreign one.
	// Usage messages are placeholders intended to be read here.
	var mode string
	flag.StringVar(&mode, "mode", "", "Work mode ('signal' or 'message')")
	var pid int
	flag.IntVar(&pid, "pid", -1, "Process identifier of the console to attach to")
	var sig int
	flag.IntVar(&sig, "sig", -1, "A control signal type")
	var msg string
	flag.StringVar(&msg, "msg", "", "A message to send to standard input")
	flag.Parse()

	switch mode {
	case "signal":
		signal.Send(pid, sig)
	case "message":
		message.Send(pid, msg)
	default:
		os.Exit(exitcodes.WrongMode)
	}
}
