//go:build windows

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/SCP002/terminator"
	"golang.org/x/sys/windows"
)

func main() {
	fmt.Print("PID to terminate: ")
	var pid int
	_, _ = fmt.Scanln(&pid)

	if err := terminator.SendSignal(pid, windows.CTRL_C_EVENT); err != nil {
		fmt.Printf("SendSignal failed with: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	terminator.WaitForProcStop(ctx, pid)
	fmt.Println("\nContinuing execution of caller")

	fmt.Print("Press <Enter> to exit...")
	_, _ = fmt.Scanln()
}
