package main

import (
	"fmt"
	"time"

	"github.com/SCP002/terminator"
)

func main() {
	fmt.Print("The PID to terminate: ")
	var pid int
	fmt.Scanln(&pid)

	opts := terminator.Options{
		Pid:          pid,
		IgnoreAbsent: false,
		Tree:         true,
		Timeout:      5000,
		Answer:       "",
	}
	err := terminator.Stop(opts)
	if err != nil {
		fmt.Println("Stop failed with:", err)
	}

	fmt.Println("Continuing execution of caller")
	time.Sleep(2 * time.Second)

	fmt.Print("Press <Enter> to exit...")
	fmt.Scanln()
}