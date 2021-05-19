@Echo Off
ChCp 65001 >Nul

Cls

:: Build
Set GOOS=windows
Set GOARCH=amd64
go build -o ..\..\assets\kamikaze.exe -ldflags -H=windowsgui kamikaze.go

Exit /B 0
