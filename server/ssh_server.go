package server

import (
	"log"
	"net"

	"ssh-chat-server/chat"

	"golang.org/x/crypto/ssh"
)

// SSHServer represents an SSH server instance
type SSHServer struct {
	config   *ssh.ServerConfig
	listener net.Listener
	port     string
}

// NewSSHServer creates a new SSH server with the given configuration
func NewSSHServer(port string, hostKey ssh.Signer) (*SSHServer, error) {
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

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	return &SSHServer{
		config:   config,
		listener: listener,
		port:     port,
	}, nil
}

// Start begins listening for SSH connections
func (s *SSHServer) Start() error {
	defer s.listener.Close()

	log.Printf("SSH server listening on port %s", s.port)
	log.Printf("Connect with: ssh -p %s localhost", s.port)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle each connection in a goroutine
		go s.handleConnection(conn)
	}
}

// handleConnection processes an incoming SSH connection
func (s *SSHServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Upgrade connection to SSH
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
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

		go s.handleSession(channel, requests)
	}
}

// handleSession processes SSH session requests
func (s *SSHServer) handleSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	// Handle session requests
	for req := range requests {
		switch req.Type {
		case "shell", "exec":
			req.Reply(true, nil)
			// Start the TUI application
			chat.RunChatTUI(channel)
			return
		case "pty-req":
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
}
