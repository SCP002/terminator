@Echo Off
ChCp 65001 >Nul

Cls

:: Build x32
Set GOOS=windows
Set GOARCH=386
go build -o .\bin\proxy_x32.exe -ldflags -H=windowsgui proxy.go

:: Build x64
Set GOOS=windows
Set GOARCH=amd64
go build -o .\bin\proxy_x64.exe -ldflags -H=windowsgui proxy.go

Exit /B 0
