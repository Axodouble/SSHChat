# Go SSH Server

A simple SSH server implementation in Go that allows clients to connect and execute basic commands.

## Features

- SSH server with password authentication
- Basic shell interface with command execution
- Support for common commands (ls, pwd, date, echo, etc.)
- Configurable port (default: 2222)

## Default Credentials

- **Username**: `admin`
- **Password**: `password123`

## Usage

### Build and Run

```bash
# Build the server
go build -o ssh-server main.go

# Run on default port (2222)
./ssh-server

# Run on custom port
./ssh-server 8022
```

### Connect to the Server

```bash
# Connect using ssh client
ssh admin@localhost -p 2222

# Enter password when prompted: password123
```

### Available Commands

Once connected, you can use these commands:

- `help` - Show available commands
- `whoami` - Show current user info
- `date` - Show current date and time
- `pwd` - Show current directory
- `ls [args]` - List directory contents
- `echo <text>` - Echo text
- `uptime` - Show system uptime
- `exit` or `quit` - Disconnect from server

You can also try other system commands available on the host.

## Security Note

This is a basic implementation for demonstration purposes. For production use, consider:

- Using public key authentication instead of passwords
- Implementing proper user management
- Adding rate limiting and connection limits
- Using persistent host keys
- Adding proper logging and monitoring
- Implementing command restrictions

## Dependencies

- `golang.org/x/crypto/ssh` - SSH protocol implementation
- `golang.org/x/crypto/ssh/terminal` - Terminal utilities
