package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sshfun/keys"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

const (
	// Default SSH port
	defaultPort = "2222"
	// Default credentials for demo purposes
	defaultUsername = "admin"
	defaultPassword = "password123"
)

func main() {
	// Generate a private key for the SSH server
	privateKey, err := keys.LoadOrGenerateHostKey(".keystore/sshHostKey.private")
	if err != nil {
		log.Fatal("Failed to generate private key: ", err)
	}

	// Configure the SSH server
	config := &ssh.ServerConfig{
		NoClientAuth: false,
		// PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		// 	// Simple password authentication
		// 	if c.User() == defaultUsername && string(pass) == defaultPassword {
		// 		log.Printf("User %s authenticated successfully", c.User())
		// 		return nil, nil
		// 	}
		// 	log.Printf("Authentication failed for user %s", c.User())
		// 	return nil, fmt.Errorf("password rejected for %q", c.User())
		// },
	}

	// Add the host key to the server config
	config.AddHostKey(privateKey)

	// Start listening on the SSH port
	port := defaultPort
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("Failed to listen on port ", port, ": ", err)
	}
	defer listener.Close()

	log.Printf("SSH server listening on port %s", port)

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle each connection in a goroutine
		go handleConnection(conn, config)
	}
}

func handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake: %v", err)
		return
	}
	defer sshConn.Close()

	log.Printf("New SSH connection from %s (user: %s)", sshConn.RemoteAddr(), sshConn.User())

	// Handle global requests
	go ssh.DiscardRequests(reqs)

	// Handle channel requests
	for newChannel := range chans {
		go handleChannel(newChannel)
	}
}

func handleChannel(newChannel ssh.NewChannel) {
	// Only accept session channels
	if newChannel.ChannelType() != "session" {
		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Printf("Could not accept channel: %v", err)
		return
	}
	defer channel.Close()

	// Handle requests for this channel
	go handleRequests(requests, channel)

	// Send welcome message
	fmt.Fprintf(channel, "Welcome to Go SSH Server!\r\n")
	fmt.Fprintf(channel, "Type 'help' for available commands or 'exit' to quit.\r\n")

	// Simple command loop
	handleShell(channel)
}

func handleRequests(requests <-chan *ssh.Request, channel ssh.Channel) {
	for req := range requests {
		switch req.Type {
		case "shell":
			// We'll handle the shell in the main channel handler
			req.Reply(true, nil)
		case "pty-req":
			// Handle PTY request
			req.Reply(true, nil)
		case "window-change":
			// Handle window resize
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
}

func handleShell(channel ssh.Channel) {
	term := term.NewTerminal(channel, "$ ")

	for {
		line, err := term.ReadLine()
		if err != nil {
			if err == io.EOF {
				return
			}
			log.Printf("Error reading line: %v", err)
			return
		}

		// Process the command
		processCommand(term, strings.TrimSpace(line))
	}
}

func processCommand(term *term.Terminal, command string) {
	if command == "" {
		return
	}

	parts := strings.Fields(command)
	cmd := parts[0]

	switch cmd {
	case "exit", "quit":
		term.Write([]byte("Goodbye!\r\n"))
		return
	case "help":
		showHelp(term)
	case "whoami":
		term.Write([]byte("You are connected to Go SSH Server\r\n"))
	case "date":
		executeSystemCommand(term, "date")
	case "pwd":
		executeSystemCommand(term, "pwd")
	case "ls":
		args := []string{"ls", "-la"}
		if len(parts) > 1 {
			args = append(args, parts[1:]...)
		}
		executeSystemCommandWithArgs(term, args)
	case "echo":
		if len(parts) > 1 {
			output := strings.Join(parts[1:], " ")
			term.Write([]byte(output + "\r\n"))
		}
	case "uptime":
		executeSystemCommand(term, "uptime")
	default:
		// Try to execute as system command
		executeSystemCommandWithArgs(term, parts)
	}
}

func showHelp(term *term.Terminal) {
	help := `Available commands:
  help     - Show this help message
  whoami   - Show current user info
  date     - Show current date and time
  pwd      - Show current directory
  ls       - List directory contents
  echo     - Echo text
  uptime   - Show system uptime
  exit     - Disconnect from server

You can also try other system commands.
`
	term.Write([]byte(help))
}

func executeSystemCommand(term *term.Terminal, command string) {
	executeSystemCommandWithArgs(term, []string{command})
}

func executeSystemCommandWithArgs(term *term.Terminal, args []string) {
	if len(args) == 0 {
		return
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		term.Write([]byte(fmt.Sprintf("Error: %v\r\n", err)))
		return
	}

	// Convert \n to \r\n for proper terminal display
	output = []byte(strings.ReplaceAll(string(output), "\n", "\r\n"))
	term.Write(output)
}

// setWinsize sets the window size for the terminal
func setWinsize(fd uintptr, w, h uint32) {
	ws := struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}{
		Row: uint16(h),
		Col: uint16(w),
	}
	syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(&ws)))
}
