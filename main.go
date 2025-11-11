package main

import (
	"log"

	"ssh-chat-server/keys"
	"ssh-chat-server/server"
)

func main() {
	// Generate or load SSH host key
	hostKey, err := keys.LoadOrGenerateHostKey(".keystore/sshHostKey.private")
	if err != nil {
		log.Fatal("Failed to load or generate host key:", err)
	}

	// Create and start SSH server
	sshServer, err := server.NewSSHServer("1234", hostKey)
	if err != nil {
		log.Fatal("Failed to create SSH server:", err)
	}

	// This will run indefinitely
	if err := sshServer.Start(); err != nil {
		log.Fatal("SSH server error:", err)
	}
}
