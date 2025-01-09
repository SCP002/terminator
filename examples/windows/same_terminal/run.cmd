@Echo Off
ChCp 65001 > Nul

Cls

Rem Download dependencies and remove unused ones
go mod tidy

Rem Build internal dependencies
PushD "..\..\..\scripts"
Call "build_windows_dependencies.cmd"
PopD

Rem Run
go run "terminator_same_terminal.go"

Exit /B 0
