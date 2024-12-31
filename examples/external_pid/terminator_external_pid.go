package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/SCP002/terminator"
)

func main() {
	fmt.Print("The PID to terminate: ")
	var pid int
	_, _ = fmt.Scanln(&pid)

	opts := terminator.Options{
		IgnoreAbsent: false,
		Tree:         true,
		Timeout:      5 * time.Second,
		Tick:         100 * time.Millisecond,
		Message:      "",
	}
	sr, err := terminator.StopThenKill(pid, opts)
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
