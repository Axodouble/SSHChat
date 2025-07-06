# Chat UI Application

This project is a simple chat user interface built in Go. It provides a basic framework for sending and displaying messages in a chat format.

## Project Structure

```
chat-ui-app
├── cmd
│   └── main.go          # Entry point of the application
├── internal
│   ├── models
│   │   ├── chat.go      # Defines the Chat struct and message management
│   │   └── message.go   # Defines the Message struct
│   ├── ui
│   │   ├── chat.go      # Logic for rendering the chat window
│   │   ├── input.go     # Manages user input for messages
│   │   └── styles.go    # Defines styles for the UI
│   └── types
│       └── types.go     # Common types and interfaces
├── go.mod               # Module definition and dependencies
├── go.sum               # Dependency checksums
└── README.md            # Project documentation
```

## Setup Instructions

1. **Clone the repository:**
   ```
   git clone <repository-url>
   cd chat-ui-app
   ```

2. **Install dependencies:**
   ```
   go mod tidy
   ```

3. **Run the application:**
   ```
   go run cmd/main.go
   ```

## Usage

- Type your message in the input area and press Enter to send.
- Messages will be displayed in the chat window.

## Contributing

Feel free to submit issues or pull requests for improvements or bug fixes.