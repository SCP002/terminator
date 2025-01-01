@Echo Off
ChCp 65001 > Nul

Cls

Rem Update dependencies
go get -u ./...

Rem Clear unused dependencies
go mod tidy

Rem Build internal dependencies
PushD "..\..\..\scripts"
Call "build_dependencies.cmd"
PopD

Rem Build x32
Set GOOS=windows
Set GOARCH=386
go build -o "terminator_external_pid_ctrlc_x32.exe" "terminator_external_pid_ctrlc.go"

Rem Build x64
Set GOOS=windows
Set GOARCH=amd64
go build -o "terminator_external_pid_ctrlc_x64.exe" "terminator_external_pid_ctrlc.go"

Exit /B 0
