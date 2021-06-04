package main

import (
	"flag"
	"os"

	"github.com/SCP002/terminator/internal/proxy/answer"
	"github.com/SCP002/terminator/internal/proxy/codes"
	"github.com/SCP002/terminator/internal/proxy/event"
)

/*
	Work modes:
	"ctrlc": Sends Ctrl + C signal to a process and terminates with it.
	"answer": Writes a message to the standard input of a process.

	Meant to be built with -ldflags -H=windowsgui build options to
	not to flash with the console during it's short life time and
	to not to call "FreeConsole" for nothing.
*/

func main() {
	// Using -h flag to display help won't work as we don't have a console and
	// attach to the foreign one. Usage messages are placeholders intended to
	// be read here.
	var mode string
	flag.StringVar(&mode, "mode", "", "Work mode ('ctrlc' or 'answer')")
	var pid int
	flag.IntVar(&pid, "pid", -1, "Process identifier of the console to attach to")
	// The message must end with a Windows newline sequence ("\r\n") to be sent.
	var msg string
	flag.StringVar(&msg, "msg", "", "A message to send to StdIn")
	flag.Parse()

	if mode == "ctrlc" {
		event.Send(pid)
	} else if mode == "answer" {
		answer.Send(pid, msg)
	} else {
		os.Exit(codes.WrongMode)
	}
}
