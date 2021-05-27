@Echo Off
ChCp 65001 >Nul

Cls

:: Build
Set GOOS=windows
Set GOARCH=amd64
go build -o ..\..\assets\send_message.exe -ldflags -H=windowsgui send_message.go

Exit /B 0
