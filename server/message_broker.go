package server

import (
	"sync"
	"time"
)

// Message represents a chat message
type Message struct {
	ID        int       `json:"id"`
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Client represents a connected chat client
type Client struct {
	Username string
	Channel  chan Message
	LastSeen int // Last message ID seen by this client
}

// MessageBroker manages all chat messages and clients
type MessageBroker struct {
	mu          sync.RWMutex
	messages    []Message
	clients     map[string]*Client
	nextID      int
	nextMessage chan Message
}

// NewMessageBroker creates a new message broker
func NewMessageBroker() *MessageBroker {
	broker := &MessageBroker{
		messages:    make([]Message, 0),
		clients:     make(map[string]*Client),
		nextID:      1,
		nextMessage: make(chan Message, 100),
	}

	// Start the message distribution goroutine
	go broker.distributeMessages()

	return broker
}

// Global message broker instance
var GlobalBroker = NewMessageBroker()

// AddClient registers a new client with the broker
func (mb *MessageBroker) AddClient(username string) *Client {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	client := &Client{
		Username: username,
		Channel:  make(chan Message, 100),
		LastSeen: 0,
	}

	mb.clients[username] = client

	// Send all existing messages to the new client
	for _, msg := range mb.messages {
		select {
		case client.Channel <- msg:
		default:
			// Channel full, skip
		}
	}

	return client
}

// RemoveClient unregisters a client from the broker
func (mb *MessageBroker) RemoveClient(username string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if client, exists := mb.clients[username]; exists {
		close(client.Channel)
		delete(mb.clients, username)
	}
}

// SendMessage adds a new message to the broker
func (mb *MessageBroker) SendMessage(sender, content string) {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	message := Message{
		ID:        mb.nextID,
		Sender:    sender,
		Content:   content,
		Timestamp: time.Now(),
	}

	mb.nextID++
	mb.messages = append(mb.messages, message)

	// Send to distribution channel
	select {
	case mb.nextMessage <- message:
	default:
		// Channel full, skip
	}
}

// distributeMessages sends new messages to all connected clients
func (mb *MessageBroker) distributeMessages() {
	for message := range mb.nextMessage {
		mb.mu.RLock()
		for _, client := range mb.clients {
			select {
			case client.Channel <- message:
			default:
				// Client channel full, skip
			}
		}
		mb.mu.RUnlock()
	}
}

// GetAllMessages returns all messages
func (mb *MessageBroker) GetAllMessages() []Message {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	// Return a copy of the messages
	result := make([]Message, len(mb.messages))
	copy(result, mb.messages)
	return result
}
