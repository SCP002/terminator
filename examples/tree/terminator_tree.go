package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/SCP002/terminator"
	"golang.org/x/sys/windows"
)

func main() {
	cmd := exec.Command("sample_top_child.cmd")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	attr := syscall.SysProcAttr{}
	attr.CreationFlags |= windows.CREATE_NEW_CONSOLE
	attr.NoInheritHandles = true
	cmd.SysProcAttr = &attr

	err := cmd.Start()
	if err != nil {
		fmt.Println("Start failed with:", err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	opts := terminator.Options{
		Pid:          cmd.Process.Pid,
		IgnoreAbsent: false,
		Tree:         true,
		Timeout:      5000,
		Answer:       "Y\r\n",
	}
	err = terminator.Stop(opts)
	if err != nil {
		fmt.Println("Stop failed with:", err)
	}

	fmt.Println("Continuing execution of caller")
	time.Sleep(2 * time.Second)

	fmt.Print("Press <Enter> to exit...")
	fmt.Scanln()
}
