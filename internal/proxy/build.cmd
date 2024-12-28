@Echo Off
ChCp 65001 > Nul

Cls

Rem Build x32
Set GOOS=windows
Set GOARCH=386
Rem Do not add "-H=windowsgui" build flag, otherwise antivirus will trigger false alarms
go build -o .\bin\proxy_x32.exe -ldflags "-s" proxy.go

Rem Build x64
Set GOOS=windows
Set GOARCH=amd64
Rem Do not add "-H=windowsgui" build flag, otherwise antivirus will trigger false alarms
go build -o .\bin\proxy_x64.exe -ldflags "-s" proxy.go

Exit /B 0
