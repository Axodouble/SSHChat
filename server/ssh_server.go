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

	// Kick users logging in as root or admin (usually bots)
	if username == "root" || username == "admin" {
		log.Printf("Rejected connection from %s: root/admin login is not allowed", sshConn.RemoteAddr())
		conn.Close()
		return
	}

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

	var tui *chat.ChatTUI
	tuiStarted := false

	// Handle session requests
	for req := range requests {
		switch req.Type {
		case "pty-req":
			req.Reply(true, nil)
		case "shell", "exec":
			req.Reply(true, nil)
			if !tuiStarted {
				// Start the TUI application with username in a goroutine
				tui = chat.NewChatTUI(channel, username)
				tuiStarted = true
				go tui.Run()
			}
		case "window-change":
			req.Reply(true, nil)
			log.Printf("Window resize event received for user: %s", username)
			// Handle terminal resize
			if tui != nil && tuiStarted {
				tui.HandleResize()
			}
		default:
			req.Reply(false, nil)
		}
	}

	// Clean up when requests channel closes
	if tui != nil {
		tui.Stop()
	}
}
