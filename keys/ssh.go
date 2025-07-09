package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

// loadOrGenerateHostKey loads an existing SSH host key or generates a new one
func LoadOrGenerateHostKey(filename string) (ssh.Signer, error) {
	// Try to load existing key
	if _, err := os.Stat(filename); err == nil {
		return loadExistingKey(filename)
	}

	// Generate new key if file doesn't exist
	return generateNewKey(filename)
}

// loadExistingKey loads an SSH host key from a file
func loadExistingKey(filename string) (ssh.Signer, error) {
	log.Printf("Loading existing host key from %s", filename)
	keyBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read host key file: %v", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from host key file")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer from key: %v", err)
	}

	return signer, nil
}

// generateNewKey creates a new SSH host key and saves it to a file
func generateNewKey(filename string) (ssh.Signer, error) {
	log.Printf("Generating new host key and saving to %s", filename)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %v", err)
	}

	// Save the key to file
	if err := saveKeyToFile(privateKey, filename); err != nil {
		return nil, err
	}

	// Convert to SSH format
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer from key: %v", err)
	}

	return signer, nil
}

// saveKeyToFile saves an RSA private key to a PEM file
func saveKeyToFile(privateKey *rsa.PrivateKey, filename string) error {
	keyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	pemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}

	// Check if the directory exists, create it if not
	if err := os.MkdirAll(".keystore", 0755); err != nil {
		return fmt.Errorf("failed to create key directory: %v", err)
	}

	keyFile, err := os.Create(filename)

	if err != nil {
		return fmt.Errorf("failed to create host key file: %v", err)
	}
	defer keyFile.Close()

	// Set restrictive permissions (readable only by owner)
	if err := keyFile.Chmod(0600); err != nil {
		return fmt.Errorf("failed to set key file permissions: %v", err)
	}

	if err := pem.Encode(keyFile, pemBlock); err != nil {
		return fmt.Errorf("failed to write host key to file: %v", err)
	}

	return nil
}
