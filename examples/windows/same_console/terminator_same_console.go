//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/SCP002/terminator"
)

func main() {
	cmd := exec.Command("ping", "-t", "127.0.0.1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Start()
	if err != nil {
		fmt.Printf("Start failed with: %v\n", err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	err = terminator.SendCtrlC(cmd.Process.Pid)
	if err != nil {
		fmt.Printf("SendCtrlC failed with: %v\n", err)
	}

	time.Sleep(2 * time.Second)
	fmt.Println("\nContinuing execution of caller")

	fmt.Print("Press <Enter> to exit...")
	_, _ = fmt.Scanln()
}
