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

:: Build x32
Set GOOS=windows
Set GOARCH=386
go build -o .\terminator_separate_batch_x32.exe terminator_separate_batch.go

:: Build x64
Set GOOS=windows
Set GOARCH=amd64
go build -o .\terminator_separate_batch_x64.exe terminator_separate_batch.go

Echo Done. Press any key to exit...
Pause >Nul
Exit /B 0
