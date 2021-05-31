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
	cmd := exec.Command("..\\sample_executable.cmd")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

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

	err = terminator.Stop(cmd.Process.Pid, false)
	if err != nil {
		fmt.Println("Stop failed with:", err)
	}

	fmt.Println("Continuing execution of caller")
	time.Sleep(2 * time.Second)

	fmt.Print("Press <Enter> to exit...")
	fmt.Scanln()
}
