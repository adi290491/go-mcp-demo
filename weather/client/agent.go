package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	model     = anthropic.ModelClaudeSonnet4_6
	maxTokens = 16000

	systemPrompt = `You are a weather assistant for US locations.
Use get_forecast with latitude/longitude (convert city names to coordinates yourself).
Use get_alerts with a two-letter US state code.`
)

// ANSI colour codes for the chat transcript.
const (
	blue   = "\u001b[94m"
	yellow = "\u001b[93m"
	green  = "\u001b[92m"
	reset  = "\u001b[0m"
)

type Agent struct {
	client   *anthropic.Client
	session  *mcp.ClientSession
	tools    []anthropic.ToolUnionParam
	getInput func() (string, bool)
}

func NewAgent(client *anthropic.Client, session *mcp.ClientSession, tools []anthropic.ToolUnionParam, getInput func() (string, bool)) *Agent {
	return &Agent{
		client:   client,
		session:  session,
		tools:    tools,
		getInput: getInput,
	}
}

// Run reads user input until EOF, answering each message with runTurn.
func (a *Agent) Run(ctx context.Context) error {
	var conversation []anthropic.MessageParam
	fmt.Println("Weather client with Claude (use 'ctrl-c' to quit)")

	for {
		fmt.Print(blue + "You" + reset + ": ")
		userInput, ok := a.getInput()
		if !ok {
			return nil
		}
		if strings.TrimSpace(userInput) == "" {
			continue
		}

		conversation = append(conversation, anthropic.NewUserMessage(anthropic.NewTextBlock(userInput)))

		var err error
		conversation, err = a.runTurn(ctx, conversation)
		if err != nil {
			return err
		}
	}
}

// runTurn calls Claude, executes any requested tools via MCP, and repeats
// until Claude stops asking for tools. It returns the extended conversation.
func (a *Agent) runTurn(ctx context.Context, conversation []anthropic.MessageParam) ([]anthropic.MessageParam, error) {
	for {
		message, err := a.runInference(ctx, conversation)
		if err != nil {
			return conversation, err
		}
		conversation = append(conversation, message.ToParam())

		var toolResults []anthropic.ContentBlockParamUnion
		for _, content := range message.Content {
			switch block := content.AsAny().(type) {
			case anthropic.TextBlock:
				fmt.Printf(yellow+"Claude"+reset+": %s\n", block.Text)
			case anthropic.ToolUseBlock:
				toolResults = append(toolResults, a.callMCPTool(ctx, block))
			}
		}

		if message.StopReason != anthropic.StopReasonToolUse {
			return conversation, nil
		}

		// All results for one assistant message go back in a single user message.
		conversation = append(conversation, anthropic.NewUserMessage(toolResults...))
	}
}

func (a *Agent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	return a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: maxTokens,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages:  conversation,
		Tools:     a.tools,
	})
}

// callMCPTool forwards a tool_use block to the MCP server and wraps the
// outcome as a tool_result. Failures are reported to Claude, not returned.
func (a *Agent) callMCPTool(ctx context.Context, block anthropic.ToolUseBlock) anthropic.ContentBlockParamUnion {
	fmt.Printf(green+"tool"+reset+": %s(%s)\n", block.Name, block.JSON.Input.Raw())

	res, err := a.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      block.Name,
		Arguments: block.Input, // raw JSON is forwarded as-is
	})
	if err != nil {
		return anthropic.NewToolResultBlock(block.ID, err.Error(), true)
	}

	return anthropic.NewToolResultBlock(block.ID, textContent(res), res.IsError)
}
