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
		fmt.Println("Start failed with:", err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	opts := terminator.Options{
		Pid:          cmd.Process.Pid,
		IgnoreAbsent: false,
		Tree:         true,
		Timeout:      5000,
		Answer:       "",
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
