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
		fmt.Println("Failed to get current workng directory:", err)
	}

	cmd := exec.Command(filepath.Join(currentDir, "sample_process_group.cmd"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	attr := syscall.SysProcAttr{}
	attr.CreationFlags |= windows.CREATE_NEW_CONSOLE
	attr.NoInheritHandles = true
	cmd.SysProcAttr = &attr

	err = cmd.Start()
	if err != nil {
		fmt.Println("Start failed with:", err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	opts := terminator.Options{
		IgnoreAbsent: false,
		Tree:         true,
		Timeout:      5 * time.Second,
		Tick:         100 * time.Millisecond,
		Answer:       "Y\r\n",
	}
	sr, err := terminator.Stop(cmd.Process.Pid, opts)
	if err != nil {
		fmt.Println("Stop failed with:", err)
	}
	prettySr, _ := json.MarshalIndent(sr, "", "    ")
	fmt.Println(string(prettySr))

	fmt.Println("Continuing execution of caller")
	time.Sleep(2 * time.Second)

	fmt.Print("Press <Enter> to exit...")
	fmt.Scanln()
}
