package main

import (
	"os/exec"
	"time"

	"github.com/SCP002/terminator"
)

func main() {
	command := exec.Command("notepad")

	err := command.Start()
	if err != nil {
		panic(err)
	}

	time.Sleep(2 * time.Second)

	err = terminator.Stop(9898, false)
	if err != nil {
		panic(err)
	}
}
