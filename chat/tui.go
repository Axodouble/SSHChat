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
	channel  ssh.Channel
	username string
	client   *ChatClient
	messages []ChatMessage
}

// NewChatTUI creates a new chat TUI instance
func NewChatTUI(channel ssh.Channel, username string) *ChatTUI {
	return &ChatTUI{
		channel:  channel,
		username: username,
		messages: make([]ChatMessage, 0),
	}
}

// RunChatTUI starts the chat terminal user interface
func RunChatTUI(channel ssh.Channel, username string) {
	tui := NewChatTUI(channel, username)
	tui.Run()
}

// Run starts the chat TUI main loop
func (c *ChatTUI) Run() {
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

	// Initial refresh to show existing messages
	c.refresh()

	// Read input from SSH channel
	buffer := make([]byte, 1024)
	currentInput := ""

	for {
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
					// Send message to broker
					GlobalChatBroker.SendMessage(c.username, currentInput)
					currentInput = ""
					// Clear the input line and redraw prompt
					c.channel.Write([]byte("\r> "))
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
				return
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
		c.refresh()
	}
}

// addMessage adds a new message to the chat and refreshes the display (legacy method, now unused)
func (c *ChatTUI) addMessage(sender, content string) {
	// This method is no longer used since messages come from the broker
}

// refresh clears the screen and redraws all messages
func (c *ChatTUI) refresh() {
	// Clear screen and move cursor to top
	c.channel.Write([]byte("\033[2J\033[H"))
	c.channel.Write([]byte("=== SSH Chat Server ===\r\n"))
	c.channel.Write([]byte(fmt.Sprintf("Connected as: %s\r\n", c.username)))
	c.channel.Write([]byte("Type your message and press Enter. Ctrl+C to quit.\r\n\r\n"))

	// Display messages
	for _, msg := range c.messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		var formattedMsg string
		if msg.Sender == "System" {
			formattedMsg = fmt.Sprintf("[%s] ** %s **\r\n", timestamp, msg.Content)
		} else {
			formattedMsg = fmt.Sprintf("[%s] %s: %s\r\n", timestamp, msg.Sender, msg.Content)
		}
		c.channel.Write([]byte(formattedMsg))
	}

	c.channel.Write([]byte("\r\n> "))
}
