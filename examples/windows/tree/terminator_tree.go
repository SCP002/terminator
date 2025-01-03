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
	"github.com/samber/lo"
	"github.com/shirou/gopsutil/v4/process"

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

	if err := cmd.Start(); err != nil {
		fmt.Printf("Start process failed with: %v\n", err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	// Get children
	children, err := terminator.FlatChildTree(cmd.Process.Pid, true)
	if err != nil {
		fmt.Printf("Get process tree failed with: %v\n", err)
	}

	// Get cmd.exe children
	cmds := lo.Filter(children, func(child *process.Process, _ int) bool {
		name, _ := child.Name()
		return name == "cmd.exe"
	})

	// Close bottom child
	if err := terminator.SendCtrlC(int(cmds[0].Pid)); err != nil {
		fmt.Printf("SendCtrlC for child process with PID %v failed with: %v\n", cmds[0].Pid, err)
	}
	if err := terminator.SendMessage(int(cmds[0].Pid), "Y\r\nexit\r\nexit\r\n"); err != nil {
		fmt.Printf("SendMessage for child process with PID %v failed with: %v\n", cmds[0].Pid, err)
	}
	// Close middle child
	if err := terminator.SendMessage(int(cmds[1].Pid), "\r\nexit\r\n"); err != nil {
		fmt.Printf("SendMessage for child process with PID %v failed with: %v\n", cmds[1].Pid, err)
	}
	// Close bottom child
	if err := terminator.SendMessage(int(cmds[2].Pid), "\r\n"); err != nil {
		fmt.Printf("SendMessage for child process with PID %v failed with: %v\n", cmds[2].Pid, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	terminator.WaitForProcStop(ctx, cmd.Process.Pid)
	fmt.Println("\nContinuing execution of caller")

	fmt.Print("Press <Enter> to exit...")
	_, _ = fmt.Scanln()
}
