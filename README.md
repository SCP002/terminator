# terminator

> Library to stop processes gracefully, even on Windows.

## Capabilities

On Windows it can:

* Send Ctrl + C to console applications
* Send Ctrl + Break to console applications
* Close graphical applications as if it's window was closed
* Send messages to standard input of console applications to answer the questions such as "Y/N?"

On Linux it can:

* Send signals (SIGINT, SIGKILL etc.) to terminal applications
* Send messages to standard input of terminal applications to answer the questions such as "Y/N?"

On macOS it can:

* Send signals (SIGINT, SIGKILL etc.) to terminal applications

Not tested on other systems.

As of go gopsutil v4.24.12 on mac OS Ventura, sending message to standard input of
terminal application returns "not implemented yet" error on attempt to get terminal from PID.

## Usage

See [examples](https://github.com/SCP002/terminator/tree/main/examples) folder and
info on [go packages](https://pkg.go.dev/github.com/SCP002/terminator).
