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
	broker   *MessageBroker
}

// NewSSHServer creates a new SSH server with the given configuration
func NewSSHServer(port string, hostKey ssh.Signer) (*SSHServer, error) {
	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}
	config.AddHostKey(hostKey)

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	// Initialize message broker
	broker := NewMessageBroker()

	// Set up the global chat broker
	chat.GlobalChatBroker = NewBrokerAdapter(broker)

	return &SSHServer{
		config:   config,
		listener: listener,
		port:     port,
		broker:   broker,
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

	username := sshConn.User()
	log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), username)

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

		go s.handleSession(channel, requests, username)
	}
}

// handleSession processes SSH session requests
func (s *SSHServer) handleSession(channel ssh.Channel, requests <-chan *ssh.Request, username string) {
	defer channel.Close()

	// Handle session requests
	for req := range requests {
		switch req.Type {
		case "shell", "exec":
			req.Reply(true, nil)
			// Start the TUI application with username
			chat.RunChatTUI(channel, username)
			return
		case "pty-req":
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
}
