//go:build !windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/SCP002/terminator"
)

func main() {
	cmd := exec.Command("ping", "127.0.0.1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		fmt.Printf("Start failed with: %v\n", err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	// Ctrl + C
	if err := terminator.SendMessage(cmd.Process.Pid, "\x03"); err != nil {
		fmt.Printf("SendMessage failed with: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	terminator.WaitForProcStop(ctx, cmd.Process.Pid)
	fmt.Println("\nContinuing execution of caller")

	fmt.Print("Press <Enter> to exit...")
	_, _ = fmt.Scanln()
}
