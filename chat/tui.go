package chat

import (
	"fmt"
	"io"
	"log"
	"time"

	"golang.org/x/crypto/ssh"
)

// Message represents a chat message
type Message struct {
	Sender    string
	Content   string
	Timestamp time.Time
}

// ChatTUI manages the terminal user interface for the chat
type ChatTUI struct {
	channel  ssh.Channel
	messages []Message
}

// NewChatTUI creates a new chat TUI instance
func NewChatTUI(channel ssh.Channel) *ChatTUI {
	return &ChatTUI{
		channel:  channel,
		messages: make([]Message, 0),
	}
}

// RunChatTUI starts the chat terminal user interface
func RunChatTUI(channel ssh.Channel) {
	tui := NewChatTUI(channel)
	tui.Run()
}

// Run starts the chat TUI main loop
func (c *ChatTUI) Run() {
	c.addMessage("System", "Welcome to the SSH Chat Server!")
	c.addMessage("Bot", "Type a message and press Enter to send")

	// Read input from SSH channel
	buffer := make([]byte, 1024)
	currentInput := ""

	for {
		n, err := c.channel.Read(buffer)
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
					c.addMessage("You", currentInput)

					// Echo response
					go func(input string) {
						time.Sleep(100 * time.Millisecond)
						c.addMessage("Bot", fmt.Sprintf("Echo: %s", input))
					}(currentInput)

					currentInput = ""
				}
			case 127, 8: // Backspace
				if len(currentInput) > 0 {
					currentInput = currentInput[:len(currentInput)-1]
					// Clear current line and redraw prompt with current input
					c.channel.Write([]byte("\r> " + currentInput + " \b"))
				}
			case 3: // Ctrl+C
				c.channel.Write([]byte("\r\nGoodbye!\r\n"))
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

// addMessage adds a new message to the chat and refreshes the display
func (c *ChatTUI) addMessage(sender, content string) {
	message := Message{
		Sender:    sender,
		Content:   content,
		Timestamp: time.Now(),
	}
	c.messages = append(c.messages, message)
	c.refresh()
}

// refresh clears the screen and redraws all messages
func (c *ChatTUI) refresh() {
	// Clear screen and move cursor to top
	c.channel.Write([]byte("\033[2J\033[H"))
	c.channel.Write([]byte("=== SSH Chat Server ===\r\n\r\n"))

	// Display messages
	for _, msg := range c.messages {
		timestamp := msg.Timestamp.Format("15:04:05")
		formattedMsg := fmt.Sprintf("[%s] %s: %s\r\n", timestamp, msg.Sender, msg.Content)
		c.channel.Write([]byte(formattedMsg))
	}

	c.channel.Write([]byte("\r\n> "))
}
