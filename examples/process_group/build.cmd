@Echo Off
ChCp 65001 >Nul

Cls

:: Update dependencies
go get -u ./...

:: Clear unused dependencies
go mod tidy

:: Build internal dependencies
PushD "..\..\scripts"
Call "build_dependencies.cmd"
PopD

:: Build x32
Set GOOS=windows
Set GOARCH=386
go build -o "terminator_process_group_x32.exe" "terminator_process_group.go"

:: Build x64
Set GOOS=windows
Set GOARCH=amd64
go build -o "terminator_process_group_x64.exe" "terminator_process_group.go"

Exit /B 0
