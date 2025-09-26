@Echo Off
ChCp 65001 > Nul

Cls

Rem Download dependencies and remove unused ones
go mod tidy

Rem Build internal dependencies
PushD "..\..\..\scripts"
Call "build_windows_dependencies.cmd"
PopD

Rem Build 386
Set GOOS=windows
Set GOARCH=386
go build -o "terminator_external_pid_ctrlc_386.exe" "terminator_external_pid_ctrlc.go"

Rem Build amd64
Set GOOS=windows
Set GOARCH=amd64
go build -o "terminator_external_pid_ctrlc_amd64.exe" "terminator_external_pid_ctrlc.go"

Rem Build arm64
Set GOOS=windows
Set GOARCH=arm64
go build -o "terminator_external_pid_ctrlc_arm64.exe" "terminator_external_pid_ctrlc.go"

Exit /B 0
