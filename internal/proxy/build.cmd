@Echo Off
ChCp 65001 > Nul

Cls

Set OldArch=%GOARCH%

Rem Build 386
Set GOOS=windows
Set GOARCH=386
go build -o .\bin\proxy_386.exe -ldflags "-s -w -H=windowsgui" proxy.go

Rem Build amd64
Set GOOS=windows
Set GOARCH=amd64
go build -o .\bin\proxy_amd64.exe -ldflags "-s -w -H=windowsgui" proxy.go

Rem Build arm64
Set GOOS=windows
Set GOARCH=arm64
go build -o .\bin\proxy_arm64.exe -ldflags "-s -w -H=windowsgui" proxy.go

Set GOARCH=%OldArch%

Exit /B 0
