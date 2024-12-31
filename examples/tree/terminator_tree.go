//go:build windows

package main

import (
	"encoding/json"
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
		fmt.Printf("Start failed with: %v\n", err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	opts := terminator.Options{
		IgnoreAbsent: false,
		Tree:         true,
		Timeout:      5 * time.Second,
		Tick:         100 * time.Millisecond,
		Message:      "Y\r\n",
	}
	sr, err := terminator.StopThenKill(cmd.Process.Pid, opts)
	if err != nil {
		fmt.Printf("Stop failed with: %v\n", err)
	}
	prettySr, _ := json.MarshalIndent(sr, "", "  ")
	fmt.Println(string(prettySr))

	fmt.Println("Continuing execution of caller")
	time.Sleep(2 * time.Second)

	fmt.Print("Press <Enter> to exit...")
	_, _ = fmt.Scanln()
}
