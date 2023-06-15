@Echo Off
ChCp 65001 >Nul

Cls

:: Build x32
Set GOOS=windows
Set GOARCH=386
:: Do not add both "-s" and "-w" build flags, otherwise antivirus will trigger false alarms
go build -o .\bin\proxy_x32.exe -ldflags "-s -H=windowsgui" proxy.go

:: Build x64
Set GOOS=windows
Set GOARCH=amd64
:: Do not add both "-s" and "-w" build flags, otherwise antivirus will trigger false alarms
go build -o .\bin\proxy_x64.exe -ldflags "-s -H=windowsgui" proxy.go

Exit /B 0
