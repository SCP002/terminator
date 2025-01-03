@Echo Off
ChCp 65001 > Nul

Cls

Rem Build x32
Set GOOS=windows
Set GOARCH=386
go build -o .\bin\proxy_x32.exe -ldflags "-s -w -H=windowsgui" proxy.go

Rem Build x64
Set GOOS=windows
Set GOARCH=amd64
go build -o .\bin\proxy_x64.exe -ldflags "-s -w -H=windowsgui" proxy.go

Exit /B 0
