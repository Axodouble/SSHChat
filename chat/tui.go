package chat

import (
	"fmt"
	"io"
	"log"
	"sort"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// ChatTUI manages the terminal user interface for the chat
type ChatTUI struct {
	channel      ssh.Channel   // SSH channel for communication
	username     string        // Username of the connected client
	client       *ChatClient   // Associated chat client
	currentInput string        // Current user input
	lastSent     time.Time     // Timestamp of last sent message
	messages     []ChatMessage // Stored chat messages
	running      bool          // Flag to indicate if TUI is running
	refreshing   bool          // Flag to prevent concurrent refreshes
	Resizing     bool          // Flag to indicate if a resize is in progress
	width        int           // Terminal width
	height       int           // Terminal height
	resizeTimer  *time.Timer   // Timer for resize debounce
	resizeMu     sync.Mutex    // Mutex for resize operations
}

// NewChatTUI creates a new chat TUI instance
func NewChatTUI(channel ssh.Channel, username string) *ChatTUI {
	return &ChatTUI{
		channel:    channel,
		username:   username,
		messages:   make([]ChatMessage, 0),
		lastSent:   time.Now(),
		running:    false,
		refreshing: false,
		Resizing:   false,
		width:      0,
		height:     0,
	}
}

// RunChatTUI starts the chat terminal user interface
func RunChatTUI(channel ssh.Channel, username string) {
	tui := NewChatTUI(channel, username)
	tui.Run()
}

// Run starts the chat TUI main loop
func (c *ChatTUI) Run() {
	c.running = true
	defer func() { c.running = false }()

	// Connect to the message broker
	if GlobalChatBroker == nil {
		c.channel.Write([]byte("Error: Message broker not available\r\n"))
		return
	}

	c.client = GlobalChatBroker.AddClient(c.username)
	defer GlobalChatBroker.RemoveClient(c.username)

	// Send welcome message
	GlobalChatBroker.SendMessage("System", fmt.Sprintf("%s joined the chat", c.username))

	// Start goroutine to handle incoming messages from broker
	go c.handleIncomingMessages()

	// Initial setup: screen refresh
	c.refresh()

	// Read input from SSH channel
	buffer := make([]byte, 1024)
	c.currentInput = ""

	for c.running {
		n, err := c.channel.Read(buffer)
		if err != nil {
			if err == io.EOF {
				log.Printf("SSH client %s disconnected", c.username)
			} else {
				log.Printf("Error reading from SSH channel for %s: %v", c.username, err)
			}
			// Send leave message
			GlobalChatBroker.SendMessage("System", fmt.Sprintf("%s left the chat", c.username))
			return
		}

		data := buffer[:n]
		for _, b := range data {
			switch b {
			case '\r', '\n': // Enter key
				if time.Since(c.lastSent) < 5000*time.Millisecond {
					continue
				}
				if c.currentInput != "" {
					// Limit input to 200 characters
					if len(c.currentInput) > 200 {
						c.currentInput = c.currentInput[:200]
					}
					// Send message to broker
					GlobalChatBroker.SendMessage(c.username, c.currentInput)
					c.lastSent = time.Now()
					c.currentInput = ""
					// Just move to new line and show prompt, let the message handler refresh
					c.channel.Write([]byte("\r\n> "))
				}
			case 127, 8: // Backspace
				if len(c.currentInput) > 0 {
					c.currentInput = c.currentInput[:len(c.currentInput)-1]
					c.fullRefresh()
				}
			case 3: // Ctrl+C
				c.channel.Write([]byte("\r\nGoodbye!\r\n"))
				GlobalChatBroker.SendMessage("System", fmt.Sprintf("%s left the chat", c.username))
				// Ensure we stop cleanly
				c.running = false
				// Close the channel to signal the SSH session to end
				c.channel.Close()
				return
			case 12: // Ctrl+L (refresh)
				c.refresh()
				if c.currentInput != "" {
					c.channel.Write([]byte(c.currentInput))
				}
			default:
				if b >= 32 && b <= 126 { // Printable characters
					if len(c.currentInput) < 200 {
						c.currentInput += string(b)
						c.channel.Write([]byte(string(b)))
					}
				}
			}
		}
	}
}

// handleIncomingMessages processes messages from the broker
func (c *ChatTUI) handleIncomingMessages() {
	for message := range c.client.Channel {
		c.messages = append(c.messages, message)
		// Add a small delay to prevent excessive refreshing during rapid message bursts
		time.Sleep(5 * time.Millisecond)
		c.fullRefresh()
	}
}

// fullRefresh redraws the screen in a resize-safe way
func (c *ChatTUI) fullRefresh() {
	c.refresh()
	c.channel.Write([]byte(c.currentInput))
}

// refresh performs a complete screen refresh in a resize-safe way
func (c *ChatTUI) refresh() {
	// Prevent concurrent refreshes
	if c.refreshing {
		return
	}
	c.refreshing = true
	defer func() { c.refreshing = false }()

	// Use the safest possible approach for clearing and redrawing
	// This sequence works reliably across different terminal types and sizes

	// Reset terminal state and clear screen
	c.channel.Write([]byte("\033c"))         // Reset terminal
	c.channel.Write([]byte("\033[2J\033[H")) // Clear screen and go to top
	c.channel.Write([]byte("\033[?25h"))     // Ensure cursor is visible

	// Draw header
	c.channel.Write([]byte(fmt.Sprintf("===== %s@cer.sh chat =====\r\n", c.username)))
	c.channel.Write([]byte("Online users: "))
	usernames := GlobalChatBroker.ListUsernames()
	sort.Strings(usernames)
	for i, user := range usernames {
		if i > 0 {
			c.channel.Write([]byte(", "))
		}
		c.channel.Write([]byte(user))
	}
	c.channel.Write([]byte("\r\n"))
	c.channel.Write([]byte("Type your message and press Enter, (5 second cooldown). Ctrl+C to quit. Ctrl+L to refresh.\r\n\r\n"))

	// Display messages (limit to last 50 to prevent screen overflow)
	messageCount := len(c.messages)
	startIdx := 0
	if messageCount > 20 {
		startIdx = messageCount - 20
	}

	for i := startIdx; i < messageCount; i++ {
		msg := c.messages[i]
		timestamp := msg.Timestamp.Format("15:04:05")
		var formattedMsg string
		if msg.Sender == "System" {
			formattedMsg = fmt.Sprintf("[%s] ** %s **\r\n", timestamp, msg.Content)
		} else {
			formattedMsg = fmt.Sprintf("[%s] %s: %s\r\n", timestamp, msg.Sender, msg.Content)
		}
		c.channel.Write([]byte(formattedMsg))
	}

	// Show prompt at the end
	c.channel.Write([]byte("> "))
}

// HandleResize handles terminal resize events
func (c *ChatTUI) HandleResize(width int, height int) {
	log.Printf("Handling resize to %dx%d", width, height)
	if !c.running {
		return
	}

	c.resizeMu.Lock()
	defer c.resizeMu.Unlock()

	// Update dimensions
	c.width = width
	c.height = height

	// Cancel existing timer if any
	if c.resizeTimer != nil {
		c.resizeTimer.Stop()
	}

	// Set a new timer to trigger refresh after resize activity settles
	c.resizeTimer = time.AfterFunc(200*time.Millisecond, func() {
		c.fullRefresh()
	})
}

// Stop gracefully stops the TUI
func (c *ChatTUI) Stop() {
	c.running = false
}
