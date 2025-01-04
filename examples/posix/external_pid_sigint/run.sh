#!/bin/bash

clear

# Download dependencies and remove unused ones
go mod tidy

# Run
go run "terminator_external_pid_sigint.go"
