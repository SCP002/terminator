@Echo Off
ChCp 65001 > Nul

Cls

Rem Download dependencies and remove unused ones
go mod tidy

Rem Build internal dependencies
PushD "..\..\..\scripts"
Call "build_dependencies.cmd"
PopD

Rem Run
go run "terminator_external_pid_ctrlc.go"

Exit /B 0
