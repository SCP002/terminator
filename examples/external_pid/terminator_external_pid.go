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
	fmt.Scanln(&pid)

	opts := terminator.Options{
		IgnoreAbsent: false,
		Tree:         true,
		Timeout:      5 * time.Second,
		Tick:         100 * time.Millisecond,
		Answer:       "",
	}
	sr, err := terminator.Stop(pid, opts)
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
