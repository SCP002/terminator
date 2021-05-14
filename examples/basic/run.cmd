@Echo Off
ChCp 65001 >Nul

Cls

:: Update dependencies
go get -u ./...
:: Clear unused dependencies
go mod tidy
:: Run
go run main.go

Exit /B 0
