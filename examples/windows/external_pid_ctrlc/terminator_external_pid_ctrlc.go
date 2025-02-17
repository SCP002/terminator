//go:build windows

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/SCP002/terminator"
)

func main() {
	fmt.Print("PID to terminate: ")
	var pid int
	_, _ = fmt.Scanln(&pid)

	if err := terminator.SendCtrlC(pid); err != nil {
		fmt.Printf("SendCtrlC failed with: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	terminator.WaitForProcStop(ctx, pid)
	fmt.Println("\nContinuing execution of caller")

	fmt.Print("Press <Enter> to exit...")
	_, _ = fmt.Scanln()
}
