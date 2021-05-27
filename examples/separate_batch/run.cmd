@Echo Off
ChCp 65001 >Nul

Cls

:: Update dependencies
go get -u ./...

:: Clear unused dependencies
go mod tidy

:: Build internal dependencies
PushD "..\..\scripts"
Call ".\build_dependencies.cmd"
PopD

:: Run
go run terminator_separate_batch.go

Exit /B 0
