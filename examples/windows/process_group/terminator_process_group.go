//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/SCP002/terminator"

	"golang.org/x/sys/windows"
)

func main() {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Failed to get current workng directory: %v\n", err)
	}

	cmd := exec.Command(filepath.Join(currentDir, "sample_process_group.cmd"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	attr := syscall.SysProcAttr{}
	attr.CreationFlags |= windows.CREATE_NEW_CONSOLE
	attr.NoInheritHandles = true
	cmd.SysProcAttr = &attr

	if err := cmd.Start(); err != nil {
		fmt.Printf("Start failed with: %v\n", err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	if err := terminator.SendSignal(cmd.Process.Pid, windows.CTRL_BREAK_EVENT); err != nil {
		fmt.Printf("SendSignal failed with: %v\n", err)
	}
	if err := terminator.SendMessage(cmd.Process.Pid, "Y\r\n"); err != nil {
		fmt.Printf("SendMessage failed with: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	terminator.WaitForProcStop(ctx, cmd.Process.Pid)
	fmt.Println("\nContinuing execution of caller")

	fmt.Print("Press <Enter> to exit...")
	_, _ = fmt.Scanln()
}
