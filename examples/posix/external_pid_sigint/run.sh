#!/bin/bash

clear

# Download dependencies and remove unused ones
go mod tidy

# Run
sudo go run "terminator_external_pid_sigint.go"
