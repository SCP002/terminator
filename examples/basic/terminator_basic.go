package main

import (
	"fmt"
	"time"

	"github.com/SCP002/terminator"
	"golang.org/x/sys/windows"
)

func main() {
	fmt.Print("The PID to terminate: ")
	var pid int
	fmt.Scanln(&pid)
	fmt.Print("Is a console application (Yes = 1 / No = 0)?: ")
	var isConsole bool
	fmt.Scanln(&isConsole)

	opts := terminator.Options{
		Pid:          pid,
		Console:      isConsole,
		Signal:       windows.CTRL_C_EVENT,
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
