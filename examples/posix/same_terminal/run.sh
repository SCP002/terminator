#!/bin/bash

clear

# Download dependencies and remove unused ones
go mod tidy

# Run
sudo go run "terminator_same_terminal.go"
