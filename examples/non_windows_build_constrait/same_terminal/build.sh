#!/bin/bash

clear

# Download dependencies and remove unused ones
go mod tidy

# Build linux 386
GOOS=linux GOARCH=386 go build -o "terminator_same_terminal_linux_386" "terminator_same_terminal.go"

# Build linux amd64
GOOS=linux GOARCH=amd64 go build -o "terminator_same_terminal_linux_amd64" "terminator_same_terminal.go"

# Build linux arm64
GOOS=linux GOARCH=arm64 go build -o "terminator_same_terminal_linux_arm64" "terminator_same_terminal.go"

# Build darwin amd64
GOOS=darwin GOARCH=amd64 go build -o "terminator_same_terminal_darwin_amd64" "terminator_same_terminal.go"

# Build darwin arm64
GOOS=darwin GOARCH=arm64 go build -o "terminator_same_terminal_darwin_arm64" "terminator_same_terminal.go"
