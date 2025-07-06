package server

import "ssh-chat-server/chat"

// BrokerAdapter adapts the server MessageBroker to the chat.Broker interface
type BrokerAdapter struct {
	broker *MessageBroker
}

// NewBrokerAdapter creates a new broker adapter
func NewBrokerAdapter(broker *MessageBroker) *BrokerAdapter {
	return &BrokerAdapter{broker: broker}
}

// AddClient implements chat.Broker interface
func (ba *BrokerAdapter) AddClient(username string) *chat.ChatClient {
	client := ba.broker.AddClient(username)

	// Convert to chat.ChatClient
	chatClient := &chat.ChatClient{
		Username: client.Username,
		Channel:  make(chan chat.ChatMessage, 100),
		LastSeen: client.LastSeen,
	}

	// Start goroutine to convert messages
	go func() {
		for msg := range client.Channel {
			chatMsg := chat.ChatMessage{
				ID:        msg.ID,
				Sender:    msg.Sender,
				Content:   msg.Content,
				Timestamp: msg.Timestamp,
			}
			select {
			case chatClient.Channel <- chatMsg:
			default:
				// Channel full, skip
			}
		}
	}()

	return chatClient
}

// RemoveClient implements chat.Broker interface
func (ba *BrokerAdapter) RemoveClient(username string) {
	ba.broker.RemoveClient(username)
}

// SendMessage implements chat.Broker interface
func (ba *BrokerAdapter) SendMessage(sender, content string) {
	ba.broker.SendMessage(sender, content)
}
