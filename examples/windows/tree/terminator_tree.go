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

	cmd := exec.Command(filepath.Join(currentDir, "sample_top_child.cmd"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	attr := syscall.SysProcAttr{}
	attr.CreationFlags |= windows.CREATE_NEW_CONSOLE
	attr.NoInheritHandles = true
	cmd.SysProcAttr = &attr

	err = cmd.Start()
	if err != nil {
		fmt.Printf("Start process failed with: %v\n", err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	// Stop children
	children, err := terminator.FlatTree(cmd.Process.Pid, false)
	if err != nil {
		fmt.Printf("Get process tree failed with: %v\n", err)
	}
	for _, child := range children {
		// Filter descendants
		if name, _ := child.Name(); name == "cmd.exe" {
			if err = terminator.SendCtrlC(int(child.Pid)); err != nil {
				fmt.Printf("SendCtrlC for PID %v failed with: %v\n", child.Pid, err)
			}
			if err = terminator.SendMessage(int(child.Pid), "Y\r\n"); err != nil {
				fmt.Printf("WriteMessage for PID %v failed with: %v\n", child.Pid, err)
			}
		}
	}

	// Stop root
	err = terminator.SendCtrlC(cmd.Process.Pid)
	if err != nil {
		fmt.Printf("SendCtrlC for PID %v failed with: %v\n", cmd.Process.Pid, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	terminator.WaitForProcStop(ctx, cmd.Process.Pid)
	fmt.Println("\nContinuing execution of caller")

	fmt.Print("Press <Enter> to exit...")
	_, _ = fmt.Scanln()
}
