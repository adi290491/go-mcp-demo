package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range tools.Tools {
		fmt.Printf("- %s: %s\n", t.Name, t.Description)
	}

	if len(os.Args) < 2 {
		return
	}
	// 2. Call a tool
	var args map[string]any
	if len(os.Args) > 2 {
		if err := json.Unmarshal([]byte(os.Args[2]), &args); err != nil {
			log.Fatalf("bad json args: %v", err)
		}
	}
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      os.Args[1],
		Arguments: args,
	})
	if err != nil {
		log.Fatalf("call tool failed: %v", err)
	}
	if res.IsError {
		fmt.Println("tool reported an error:")
	}
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			fmt.Println(tc.Text)
		}
	}
}
