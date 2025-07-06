package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

func main() {
	// Generate or load SSH host key
	hostKey, err := generateHostKey()
	if err != nil {
		log.Fatal("Failed to generate host key:", err)
	}

	// Configure SSH server
	config := &ssh.ServerConfig{
		// Allow any user to connect (you can add authentication here)
		NoClientAuth: true,
		// Or use password authentication:
		// PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		//     if c.User() == "user" && string(pass) == "password" {
		//         return nil, nil
		//     }
		//     return nil, fmt.Errorf("password rejected for %q", c.User())
		// },
	}
	config.AddHostKey(hostKey)

	// Start SSH server
	listener, err := net.Listen("tcp", ":2222")
	if err != nil {
		log.Fatal("Failed to listen on port 2222:", err)
	}
	defer listener.Close()

	log.Println("SSH server listening on port 2222")
	log.Println("Connect with: ssh -p 2222 localhost")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle each connection in a goroutine
		go handleSSHConnection(conn, config)
	}
}

func generateHostKey() (ssh.Signer, error) {
	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Convert to SSH format
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, err
	}

	return signer, nil
}

func handleSSHConnection(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	// Upgrade connection to SSH
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("Failed to handshake: %v", err)
		return
	}
	defer sshConn.Close()

	log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.User())

	// Handle global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Failed to accept channel: %v", err)
			continue
		}

		go handleSession(channel, requests)
	}
}

func handleSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	// Handle session requests
	for req := range requests {
		switch req.Type {
		case "shell", "exec":
			req.Reply(true, nil)
			// Start the TUI application
			runChatTUI(channel)
			return
		case "pty-req":
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
}

func runChatTUI(channel ssh.Channel) {
	// Create a custom terminal interface using the SSH channel
	// We'll use a simpler approach that writes directly to the channel

	var messages []string

	addMessage := func(sender, message string) {
		timestamp := time.Now().Format("15:04:05")
		formattedMsg := fmt.Sprintf("[%s] %s: %s\r\n", timestamp, sender, message)
		messages = append(messages, formattedMsg)

		// Clear screen and redraw
		channel.Write([]byte("\033[2J\033[H")) // Clear screen and move cursor to top
		channel.Write([]byte("=== SSH Chat Server ===\r\n\r\n"))

		// Display messages
		for _, msg := range messages {
			channel.Write([]byte(msg))
		}

		channel.Write([]byte("\r\n> "))
	}

	addMessage("System", "Welcome to the SSH Chat Server!")
	addMessage("Bot", "Type a message and press Enter to send")

	// Read input from SSH channel
	buffer := make([]byte, 1024)
	currentInput := ""

	for {
		n, err := channel.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Println("SSH client disconnected")
			} else {
				log.Printf("Error reading from SSH channel: %v", err)
			}
			return
		}

		data := buffer[:n]
		for _, b := range data {
			switch b {
			case '\r', '\n': // Enter key
				if currentInput != "" {
					addMessage("You", currentInput)

					// Echo response
					go func(input string) {
						time.Sleep(100 * time.Millisecond)
						addMessage("Bot", fmt.Sprintf("Echo: %s", input))
					}(currentInput)

					currentInput = ""
				}
			case 127, 8: // Backspace
				if len(currentInput) > 0 {
					currentInput = currentInput[:len(currentInput)-1]
					// Clear current line and redraw prompt with current input
					channel.Write([]byte("\r> " + currentInput + " \b"))
				}
			case 3: // Ctrl+C
				channel.Write([]byte("\r\nGoodbye!\r\n"))
				return
			default:
				if b >= 32 && b <= 126 { // Printable characters
					currentInput += string(b)
					channel.Write([]byte(string(b)))
				}
			}
		}
	}
}
