package chat

import (
	"fmt"
	"io"
	"log"
	"time"

	"golang.org/x/crypto/ssh"
)

// We'll define our own Message type here to avoid circular imports
// This should match the server.Message type

// Message represents a chat message (kept for backward compatibility)
type Message struct {
	Sender    string
	Content   string
	Timestamp time.Time
}

// ChatTUI manages the terminal user interface for the chat
type ChatTUI struct {
	channel     ssh.Channel
	username    string
	client      *ChatClient
	messages    []ChatMessage
	headerLines int  // Number of lines used for the header
	needsRedraw bool // Flag to indicate if full redraw is needed
	running     bool // Flag to indicate if TUI is running
	refreshing  bool // Flag to prevent concurrent refreshes
}

// NewChatTUI creates a new chat TUI instance
func NewChatTUI(channel ssh.Channel, username string) *ChatTUI {
	return &ChatTUI{
		channel:     channel,
		username:    username,
		messages:    make([]ChatMessage, 0),
		headerLines: 4, // Header takes 4 lines: title, username, instructions, blank line
		needsRedraw: true,
		running:     false,
		refreshing:  false,
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

	// Initial setup: full screen refresh
	c.fullRefresh()

	// Read input from SSH channel
	buffer := make([]byte, 1024)
	currentInput := ""

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
				if currentInput != "" {
					// Limit input to 200 characters
					if len(currentInput) > 200 {
						currentInput = currentInput[:200]
					}
					// Send message to broker
					GlobalChatBroker.SendMessage(c.username, currentInput)
					currentInput = ""
					// Just move to new line and show prompt, let the message handler refresh
					c.channel.Write([]byte("\r\n> "))
				}
			case 127, 8: // Backspace
				if len(currentInput) > 0 {
					currentInput = currentInput[:len(currentInput)-1]
					// Clear current line and redraw prompt with current input
					c.channel.Write([]byte("\r> " + currentInput + " \b"))
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
				c.fullRefresh()
				if currentInput != "" {
					c.channel.Write([]byte(currentInput))
				}
			default:
				if b >= 32 && b <= 126 { // Printable characters
					currentInput += string(b)
					c.channel.Write([]byte(string(b)))
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
		time.Sleep(50 * time.Millisecond)
		c.refresh()
	}
}

// refresh redraws the screen in a resize-safe way
func (c *ChatTUI) refresh() {
	// Simply do a full refresh - it's more reliable than trying to preserve positioning
	c.fullRefresh()
}

// fullRefresh performs a complete screen refresh in a resize-safe way
func (c *ChatTUI) fullRefresh() {
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
	c.channel.Write([]byte("=== cer.sh chat ===\r\n"))
	c.channel.Write([]byte(fmt.Sprintf("Connected as: %s\r\n", c.username)))
	c.channel.Write([]byte("Type your message and press Enter. Ctrl+C to quit. Ctrl+L to refresh.\r\n\r\n"))

	// Display messages (limit to last 50 to prevent screen overflow)
	messageCount := len(c.messages)
	startIdx := 0
	if messageCount > 50 {
		startIdx = messageCount - 50
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
func (c *ChatTUI) HandleResize() {
	if !c.running {
		return
	}
	// Add a small delay to ensure the terminal has finished resizing
	time.Sleep(500 * time.Millisecond)

	// Trigger a full refresh when the terminal is resized
	c.fullRefresh()
}

// Stop gracefully stops the TUI
func (c *ChatTUI) Stop() {
	c.running = false
}
