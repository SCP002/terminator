@Echo Off
ChCp 65001 > Nul

Cls

Rem Download dependencies and remove unused ones
go mod tidy

Rem Build internal dependencies
PushD "..\..\..\scripts"
Call "build_dependencies.cmd"
PopD

Rem Build 386
Set GOOS=windows
Set GOARCH=386
go build -o "terminator_tree_386.exe" "terminator_tree.go"

Rem Build amd64
Set GOOS=windows
Set GOARCH=amd64
go build -o "terminator_tree_amd64.exe" "terminator_tree.go"

Exit /B 0
