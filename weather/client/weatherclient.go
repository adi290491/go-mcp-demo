package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/joho/godotenv"
)

func main() {
	serverPath := flag.String("server", "./weather", "path to the weather MCP server binary")
	flag.Parse()

	apiKey, err := loadAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	ctx := context.Background()
	session, err := connectMCP(ctx, *serverPath)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, t := range listed.Tools {
		fmt.Printf("- %s: %s\n", t.Name, t.Description)
	}

	tools, err := mcpToolsToAnthropic(listed.Tools)
	if err != nil {
		log.Fatal(err)
	}

	agent := NewAgent(&client, session, tools, stdinReader())
	if err := agent.Run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// stdinReader returns a function that yields one line of stdin per call.
func stdinReader() func() (string, bool) {
	scanner := bufio.NewScanner(os.Stdin)
	return func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}
}

// loadAPIKey reads ANTHROPIC_API_KEY from the environment or a local .env file.
func loadAPIKey() (string, error) {
	if err := godotenv.Load(); err != nil {
		return "", fmt.Errorf("load .env file: %w", err)
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	switch {
	case apiKey == "":
		return "", errors.New("ANTHROPIC_API_KEY is not set - add it to .env or the environment")
	case strings.TrimSpace(apiKey) != apiKey:
		return "", errors.New("ANTHROPIC_API_KEY has leading or trailing whitespace")
	case !strings.HasPrefix(apiKey, "sk-ant-"):
		return "", errors.New("ANTHROPIC_API_KEY doesn't start with sk-ant- - check you're using an Anthropic key")
	}
	return apiKey, nil
}
