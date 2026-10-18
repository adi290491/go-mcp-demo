package main

import (
	"context"
	"log"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()

	// Create a new client
	client := mcp.NewClient(&mcp.Implementation{Name: "weather-client", Version: "v1.0.0"}, nil)

	// connect to a server over stdin/stdout
	transport := &mcp.CommandTransport{
		Command: exec.Command("./weather"),
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	// call the tool

}
