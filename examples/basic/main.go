package main

import (
	"context"
	"fmt"
	"log"

	hawksdk "github.com/GrayCodeAI/hawk-sdk-go"
)

func main() {
	client := hawksdk.New()

	// Health check
	health, err := client.Health(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Hawk daemon: version=%s, sessions=%d\n", health.Version, health.Sessions)

	// Chat
	resp, err := client.Chat(context.Background(), hawksdk.ChatRequest{
		Prompt: "Explain what a closure is in Go",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Response: %s\n", resp.Response)
}
