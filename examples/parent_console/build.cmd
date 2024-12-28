@Echo Off
ChCp 65001 > Nul

Cls

Rem Update dependencies
go get -u ./...

Rem Clear unused dependencies
go mod tidy

Rem Build internal dependencies
PushD "..\..\scripts"
Call "build_dependencies.cmd"
PopD

Rem Build x32
Set GOOS=windows
Set GOARCH=386
go build -o "terminator_parent_console_x32.exe" "terminator_parent_console.go"

Rem Build x64
Set GOOS=windows
Set GOARCH=amd64
go build -o "terminator_parent_console_x64.exe" "terminator_parent_console.go"

Exit /B 0
