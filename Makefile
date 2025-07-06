# SSH Chat Server Makefile

.PHONY: all build run clean test help

# Default target
all: build

# Build the application
build:
	@echo "Building SSH Chat Server..."
	go build -o bin/ssh-chat-server .

# Build for different platforms
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/ssh-chat-server-linux .

build-windows:
	@echo "Building for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/ssh-chat-server-windows.exe .

build-mac:
	@echo "Building for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/ssh-chat-server-mac .

# Build for all platforms
build-all: build-linux build-windows build-mac

# Run the application
run:
	@echo "Starting SSH Chat Server..."
	go run .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f ssh-chat-server
	rm -rf bin/
	rm -f ssh_host_key

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Update dependencies
deps:
	@echo "Updating dependencies..."
	go mod tidy
	go mod download

# Display help
help:
	@echo "Available targets:"
	@echo "  build       - Build the application"
	@echo "  run         - Run the application"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  deps        - Update dependencies"
	@echo "  build-all   - Build for all platforms"
	@echo "  help        - Show this help message"
