package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type echoInput struct {
	Msg string `json:"msg" jsonschema:"text to echo"`
}

// newTestSession runs an in-memory MCP server with an "echo" tool and a
// "fail" tool, and returns a connected client session.
func newTestSession(t *testing.T) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "v0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "echo", Description: "Echo a message"},
		func(ctx context.Context, req *mcp.CallToolRequest, in echoInput) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: "echo: " + in.Msg}}}, nil, nil
		})
	mcp.AddTool(server, &mcp.Tool{Name: "fail", Description: "Always fails"},
		func(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, any, error) {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "boom"}}}, nil, nil
		})

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0"}, nil).
		Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

// fakeClaude replays canned Messages API responses in order and records
// each request body.
func fakeClaude(t *testing.T, responses ...string) (*anthropic.Client, *[]map[string]any) {
	t.Helper()
	var requests []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		requests = append(requests, req)
		if len(requests) > len(responses) {
			t.Errorf("unexpected request #%d", len(requests))
			http.Error(w, "no more responses", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, responses[len(requests)-1])
	}))
	t.Cleanup(srv.Close)

	client := anthropic.NewClient(option.WithBaseURL(srv.URL), option.WithAPIKey("test"), option.WithMaxRetries(0))
	return &client, &requests
}

func message(stopReason, content string) string {
	return fmt.Sprintf(`{"id":"msg","type":"message","role":"assistant","model":"m",
		"content":[%s],"stop_reason":%q,"usage":{"input_tokens":1,"output_tokens":1}}`, content, stopReason)
}

func TestRunTurnExecutesToolsViaMCP(t *testing.T) {
	session := newTestSession(t)
	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	tools, err := mcpToolsToAnthropic(listed.Tools)
	if err != nil {
		t.Fatal(err)
	}

	client, requests := fakeClaude(t,
		// Turn 1: Claude asks for two tools in parallel.
		message("tool_use", `
			{"type":"text","text":"Checking"},
			{"type":"tool_use","id":"toolu_1","name":"echo","input":{"msg":"hi"}},
			{"type":"tool_use","id":"toolu_2","name":"fail","input":{}}`),
		// Turn 2: Claude answers.
		message("end_turn", `{"type":"text","text":"Done"}`),
	)

	agent := NewAgent(client, session, tools, nil)
	conversation := []anthropic.MessageParam{anthropic.NewUserMessage(anthropic.NewTextBlock("go"))}
	conversation, err = agent.runTurn(context.Background(), conversation)
	if err != nil {
		t.Fatal(err)
	}

	// user, assistant(tool_use), user(tool_results), assistant(text)
	if len(conversation) != 4 {
		t.Fatalf("want 4 messages, got %d", len(conversation))
	}
	if len(*requests) != 2 {
		t.Fatalf("want 2 API calls, got %d", len(*requests))
	}

	// The second request must carry both tool results in a single user message.
	msgs := (*requests)[1]["messages"].([]any)
	last := msgs[len(msgs)-1].(map[string]any)
	if last["role"] != "user" {
		t.Fatalf("last message role = %v, want user", last["role"])
	}
	results := last["content"].([]any)
	if len(results) != 2 {
		t.Fatalf("want 2 tool results in one message, got %d", len(results))
	}

	want := []struct {
		id, text string
		isError  bool
	}{
		{"toolu_1", "echo: hi", false},
		{"toolu_2", "boom", true},
	}
	for i, w := range want {
		r := results[i].(map[string]any)
		text := r["content"].([]any)[0].(map[string]any)["text"]
		isError, _ := r["is_error"].(bool)
		if r["type"] != "tool_result" || r["tool_use_id"] != w.id || text != w.text || isError != w.isError {
			t.Errorf("result %d = %v, want id=%s text=%q is_error=%v", i, r, w.id, w.text, w.isError)
		}
	}
}

func TestMCPToolsToAnthropic(t *testing.T) {
	listed, err := newTestSession(t).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	tools, err := mcpToolsToAnthropic(listed.Tools)
	if err != nil {
		t.Fatal(err)
	}

	byName := map[string]*anthropic.ToolParam{}
	for _, tool := range tools {
		byName[tool.OfTool.Name] = tool.OfTool
	}
	echo := byName["echo"]
	if echo == nil {
		t.Fatalf("echo tool missing from %v", byName)
	}
	if echo.Description.Value != "Echo a message" {
		t.Errorf("description = %q", echo.Description.Value)
	}
	if _, ok := echo.InputSchema.Properties.(map[string]any)["msg"]; !ok {
		t.Errorf("schema missing msg property: %v", echo.InputSchema.Properties)
	}
	if len(echo.InputSchema.Required) != 1 || echo.InputSchema.Required[0] != "msg" {
		t.Errorf("required = %v, want [msg]", echo.InputSchema.Required)
	}
}
