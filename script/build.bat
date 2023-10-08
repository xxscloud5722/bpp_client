@echo off

cd ../
go env -w CGO_ENABLED=0

@rem Windows
go env -w GOOS=windows
cd ./src
go mod tidy
go build -o ../dist/bpp.exe main.go

@rem Linux
go env -w GOOS=linux
go build -o ../dist/bpp main.go