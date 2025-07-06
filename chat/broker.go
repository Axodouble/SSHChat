package chat

import "time"

// Message represents a chat message
type ChatMessage struct {
	ID        int
	Sender    string
	Content   string
	Timestamp time.Time
}

// ChatClient represents a connected chat client
type ChatClient struct {
	Username string
	Channel  chan ChatMessage
	LastSeen int
}

// Broker interface to interact with the message broker
type Broker interface {
	AddClient(username string) *ChatClient
	RemoveClient(username string)
	SendMessage(sender, content string)
}

// Global broker instance will be set by the server
var GlobalChatBroker Broker
