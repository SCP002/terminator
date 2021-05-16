package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/SCP002/terminator"
)

func main() {
	cmd := exec.Command("..\\..\\assets\\sample-executable.cmd")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// cmd.Stdin = os.Stdin

	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	err = terminator.Stop(cmd.Process.Pid, false)
	if err != nil {
		fmt.Println("Stop failed with:", err)
	}

	fmt.Println("Continuing execution")
	time.Sleep(2 * time.Second)

	fmt.Print("Press <Enter> to exit...")
	fmt.Scanln()
}
