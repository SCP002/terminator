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

	wnd, err := terminator.GetMainWindow(pid, false)
	if err != nil {
		fmt.Printf("GetMainWindow failed with: %v\n", err)
	}
	err = terminator.CloseWindow(wnd, false)
	if err != nil {
		fmt.Printf("CloseWindow failed with: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	terminator.WaitForProcStop(ctx, pid)
	fmt.Println("\nContinuing execution of caller")

	fmt.Print("Press <Enter> to exit...")
	_, _ = fmt.Scanln()
}
