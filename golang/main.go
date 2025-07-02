package main

import (
	"crypto/sha256"
	"fmt"
	"time"
)

func main() {
	text := "123abc"

	// Get the starting date and time
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		fmt.Printf("\nTime taken: %s\n", elapsed)
	}()

	iteration := 0
	for {
		candidate := fmt.Sprintf("%s %d", text, iteration)
		sha256Hash := sha256.Sum256([]byte(candidate))
		hashStr := fmt.Sprintf("%x", sha256Hash)
		if hashStr[:6] == "000000" {
			fmt.Printf("\nFound: %s\n", candidate)
			fmt.Printf("SHA256: %s\n", hashStr)
			break
		}
		iteration++
	}
}
