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

:: Run
go run "terminator_parent_console.go"

Exit /B 0
