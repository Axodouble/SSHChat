package main

import (
	"log"
	"math/rand"
	"net"
	"os"
	"sshfun/keys"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	defaultPort = "2222"
)

func main() {
	// Generate a private key for the SSH server
	privateKey, err := keys.LoadOrGenerateHostKey(".keystore/sshHostKey.private")
	if err != nil {
		log.Fatal("Failed to generate private key: ", err)
	}

	// Configure the SSH server
	config := &ssh.ServerConfig{
		NoClientAuth: true,
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

	// Clear screen and reset cursor position (like htop)
	channel.Write([]byte("\033[2J\033[1;1H\033[?25h"))

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
	// Initialize random seed based on current time
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Configuration for the tarpit
	const (
		minDelay      = 5000  // 5 seconds minimum delay
		maxDelay      = 15000 // 15 seconds maximum delay
		minLineLength = 10
		maxLineLength = 80
	)

	log.Printf("Starting SSH tarpit session")

	// Keep sending random banner lines indefinitely
	for {
		// Generate a random delay between messages (in milliseconds)
		delayMs := minDelay + rng.Intn(maxDelay-minDelay)

		// Sleep for the delay period
		time.Sleep(time.Duration(delayMs) * time.Millisecond)

		// Generate a random banner line
		line := generateRandomBannerLine(rng, minLineLength, maxLineLength)

		// Try to write the line to the channel
		_, err := channel.Write([]byte(line + "\r\n"))
		if err != nil {
			log.Printf("Client disconnected: %v", err)
			return
		}

		log.Printf("Sent tarpit line: %s", line)
	}
}

// generateRandomBannerLine creates a random SSH banner-like line
func generateRandomBannerLine(rng *rand.Rand, minLen, maxLen int) string {
	// Random length between min and max
	length := minLen + rng.Intn(maxLen-minLen)

	// Character set for random lines (printable ASCII excluding some problematic chars)
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.-_"

	line := make([]byte, length)
	for i := 0; i < length; i++ {
		line[i] = chars[rng.Intn(len(chars))]
	}

	// Make sure it doesn't start with "SSH-" as that would be a valid SSH banner
	result := string(line)
	if len(result) >= 4 && result[:4] == "SSH-" {
		result = "XXX-" + result[4:]
	}

	return result
}
