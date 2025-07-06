all: build-all

build:
	go build -o ./bin/main-native

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/main-linux

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ./bin/main-windows.exe

build-webassembly:
	CGO_ENABLED=0 GOOS=js GOARCH=wasm go build -o ./bin/main.wasm

build-all: build-linux build-windows build-webassembly

.PHONY: all build build-linux build-windows build-webassembly build-all
