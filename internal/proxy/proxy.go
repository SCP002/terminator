package main

import (
	"flag"
	"os"

	"github.com/SCP002/terminator/internal/proxy/codes"
	"github.com/SCP002/terminator/internal/proxy/event"
	"github.com/SCP002/terminator/internal/proxy/message"
)

/*
	Work modes:
	"ctrlc": recieves PID of the process to send Ctrl + C signal to,
	and terminates together with the target process.
	"msg": recieves PID of the process and a string message to write
	to standard input of that process.

	Meant to be built with -ldflags -H=windowsgui build options to
	not to flash with the console during it's short life time and
	to not to call "FreeConsole" for nothing.
*/

func main() {
	// Using -h flag to display help won't work as we don't have a console and
	// attach to the foreign one. Usage messages are placeholders intended to
	// be read here.
	var mode string
	flag.StringVar(&mode, "mode", "", "Work mode ('ctrlc' or 'msg')")
	var pid int
	flag.IntVar(&pid, "pid", -1, "Process identifier of the console to attach to")
	// Message should end with a Windows newline sequence ("\r\n") to be sent.
	var msg string
	flag.StringVar(&msg, "msg", "", "Message to send to StdIn")
	flag.Parse()

	if mode == "ctrlc" {
		event.Send(pid)
	} else if mode == "msg" {
		message.Send(pid, msg)
	} else {
		os.Exit(codes.WrongMode)
	}
}
