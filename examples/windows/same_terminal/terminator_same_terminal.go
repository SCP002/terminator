//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/SCP002/terminator"
	"golang.org/x/sys/windows"
)

func main() {
	cmd := exec.Command("ping", "-t", "127.0.0.1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		fmt.Printf("Start failed with: %v\n", err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	if err := terminator.SendSignal(cmd.Process.Pid, windows.CTRL_C_EVENT); err != nil {
		fmt.Printf("SendSignal failed with: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	terminator.WaitForProcStop(ctx, cmd.Process.Pid)
	fmt.Println("\nContinuing execution of caller")

	fmt.Print("Press <Enter> to exit...")
	_, _ = fmt.Scanln()
}
