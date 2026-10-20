package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "weather",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_forecast",
		Description: "Get the weather forecast for a US location by latitude and longitude",
	}, getForecast)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_alerts",
		Description: "Get active weather alerts for a US state",
	}, getAlerts)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
