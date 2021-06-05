package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/SCP002/terminator"
)

func main() {
	cmd := exec.Command("..\\sample_executable.cmd")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		fmt.Println("Start failed with:", err)
	}

	fmt.Println("Process started")
	time.Sleep(2 * time.Second)

	opts := terminator.Options{
		Pid:          cmd.Process.Pid,
		Console:      true,
		IgnoreAbsent: false,
		Tree:         true,
		Timeout:      5000,
		// If a batch executable runs in the same console as a caller,
		// prompt is skipped automatically (no need for an answer).
		// If this program itself is launched from a batch file
		// (e.g. run.cmd), prompt appears After this program ends,
		// thus answering is beyond the scope of this program
		// (no sense in answering).
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
