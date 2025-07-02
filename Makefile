# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
BINARY_NAME=main
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe

# Default target
all: build

# Build for current platform
build:
	$(GOBUILD) -o ./bin/$(BINARY_NAME) -v ./...

# Clean
clean:
	$(GOCLEAN)
	rm -rf ./bin

# Run
run:
	$(GOBUILD) -o ./bin/$(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o ./bin/$(BINARY_UNIX) -v

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o ./bin/$(BINARY_WINDOWS) -v

# Build for both platforms
build-cross: build-linux build-windows

# Build everything
build-all: build-cross

# Install dependencies
deps:
	$(GOGET) -d -v ./...

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build for current platform"
	@echo "  build-linux   - Build for Linux (amd64)"
	@echo "  build-windows - Build for Windows (amd64)"
	@echo "  build-cross   - Build for both Linux and Windows"
	@echo "  build-all     - Build for current platform + cross compile"
	@echo "  clean         - Clean build artifacts"
	@echo "  run           - Build and run the application"
	@echo "  deps          - Download dependencies"
	@echo "  help          - Show this help message"

.PHONY: all build test clean run build-linux build-windows build-cross build-all deps help
