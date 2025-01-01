//go:build windows

package main

import (
	"fmt"
	"time"

	"github.com/SCP002/terminator"
)

func main() {
	fmt.Print("PID to terminate: ")
	var pid int
	_, _ = fmt.Scanln(&pid)

	err := terminator.SendCtrlC(pid)
	if err != nil {
		fmt.Printf("SendCtrlC failed with: %v\n", err)
	}

	fmt.Println("Continuing execution of caller")
	time.Sleep(2 * time.Second)

	fmt.Print("Press <Enter> to exit...")
	_, _ = fmt.Scanln()
}
