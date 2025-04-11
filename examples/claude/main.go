// Package main provides a runnable example demonstrating the toolkit with Claude.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/joho/godotenv"

	// Toolkit framework
	"github.com/hamzaessahbaoui/ai-toolkit/toolkit"

	// Implementation packages (core logic + types)
	"github.com/hamzaessahbaoui/ai-toolkit/pkg/tools/operations"
	"github.com/hamzaessahbaoui/ai-toolkit/pkg/tools/response"
	"github.com/hamzaessahbaoui/ai-toolkit/pkg/tools/search"
)

// Define the Claude Client struct with Client, Params, and Toolkit
type ClaudeClient struct {
	Client  *anthropic.Client
	Params  *anthropic.MessageNewParams
	Toolkit *toolkit.Toolkit
}

func main() {
	// Load .env file, ignore error if it doesn't exist
	_ = godotenv.Load()

	// get anthropic api key from env
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: ANTHROPIC_API_KEY environment variable is required.")
		fmt.Println("Please create a .env file with ANTHROPIC_API_KEY=your_key")
		os.Exit(1)
	}

	// Define handler wrappers for implementation logic for the toolkit

	handleEditFile := func(ctx context.Context, args operations.EditFileArgs) (interface{}, error) {
		return operations.EditFile(ctx, args)
	}
	handleReadFile := func(ctx context.Context, args operations.ReadFileArgs) (interface{}, error) {
		return operations.ReadFile(ctx, args)
	}
	handleSearchWeb := func(ctx context.Context, args search.SearchWebArgs) (interface{}, error) {
		return search.SearchWeb(ctx, args)
	}
	handleFetchURLContent := func(ctx context.Context, args search.FetchURLArgs) (interface{}, error) {
		return search.FetchURLContent(ctx, args)
	}
	handleModelThinking := func(ctx context.Context, args response.ModelThinkingArgs) (interface{}, error) {
		return response.LogThinking(ctx, args)
	}
	handleModelResponse := func(ctx context.Context, args response.ModelResponseArgs) (interface{}, error) {
		return response.LogResponse(ctx, args)
	}

	// Use toolkit builders to define the toolkit structure
	// start with the parents
	opsParent := toolkit.NewParent(
		"operations",
		"Handles file system tasks like reading and editing files.",
		toolkit.NewChild("edit_file", "Writes content to a file.", handleEditFile),
		toolkit.NewChild("read_file", "Reads content from a file.", handleReadFile),
	)
	searchParent := toolkit.NewParent(
		"search",
		"Handles web searches and fetching content from URLs.",
		toolkit.NewChild("search_web", "Performs a web search (mocked).", handleSearchWeb),
		toolkit.NewChild("fetch_url_content", "Fetches content from a URL (mocked).", handleFetchURLContent),
	)
	respParent := toolkit.NewParent(
		"response",
		"Handles showing the model thinking and final responses.",
		toolkit.NewChild("model_thinking", "Log the model's thinking to the user.", handleModelThinking),
		toolkit.NewChild("model_response", "Log the model's response to the user.", handleModelResponse),
	)
	// note: you can add more parents and children to the toolkit

	// Create and assign the main toolkit instance for this example
	tkInstance := toolkit.New(
		"example_toolkit", // Specific name for the toolkit
		opsParent,
		searchParent,
		respParent,
	)

	client := NewClaudeClient(apiKey, tkInstance)

	// define a prompt
	prompt := "Read the file 'test.txt', then search the web for best practices about ai tool calling, then write a new file 'output.txt' with dummy content --all in one invocation"

	fmt.Println("--- Starting Conversation ---")
	finalResult, err := client.GenerateContent(prompt)
	if err != nil {
		fmt.Println("\n--- Conversation Error ---")
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("\n--- Final Result ---")
	fmt.Println(finalResult)
}
func NewClaudeClient(apiKey string, toolkit *toolkit.Toolkit) *ClaudeClient {
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable not set.")
	}
	params := anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.ModelClaude3_7Sonnet20250219),
		MaxTokens: anthropic.Int(1000),
		System: anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(
				`You are a helpful assistant.
				You can execute multiple tools in one invocation.
				You always think first. and to give a response to the user, you have to use the right tool.`),
		}),
		Tools:       anthropic.F([]anthropic.ToolUnionUnionParam{GetToolkit(toolkit)}),
		Temperature: anthropic.Float(0.5),
		ToolChoice: anthropic.F[anthropic.ToolChoiceUnionParam](anthropic.ToolChoiceAutoParam{
			Type: anthropic.F(anthropic.ToolChoiceAutoTypeAuto),
		}),
	}
	Client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &ClaudeClient{Client: Client, Params: &params, Toolkit: toolkit}
}

// GenerateContent initiates and manages the conversation loop with Claude.
func (c *ClaudeClient) GenerateContent(prompt string) (string, error) {

	message := anthropic.NewUserMessage(
		anthropic.NewTextBlock(prompt),
	)
	history := []anthropic.MessageParam{message}
	ctx := context.Background()
	var finalThinking, finalMessageResponse string

	// Conversation Loop - Max 5 turns to prevent infinite loops
	for i := 0; i < 5; i++ {
		c.Params.Messages = anthropic.F(history)
		fmt.Printf("--- Calling Claude API (Turn %d) ---\n", i+1)
		response, err := c.Client.Messages.New(ctx, *c.Params)
		if err != nil {
			fmt.Println("Error calling Claude API:", err)
			return "", err
		}
		fmt.Println("--- Usage ---")
		fmt.Println("Input tokens:", response.Usage.InputTokens)
		fmt.Println("Output tokens:", response.Usage.OutputTokens)
		history = append(history, response.ToParam())

		toolUsedInTurn := false
		var thinkingInTurn, messageInTurn string

		for _, block := range response.Content {
			switch b := block.AsUnion().(type) {
			case anthropic.TextBlock:
				messageInTurn = b.Text
			case anthropic.ThinkingBlock:
				thinkingInTurn = b.Thinking
			case anthropic.ToolUseBlock:
				fmt.Println("Tool Used:", b.Name)
				toolUsedInTurn = true
				var toolResultBlock anthropic.ToolResultBlockParam

				toolkitResponse, toolErr := c.Toolkit.HandleToolKit(ctx, b.Input)
				toolResult, marshalErr := json.Marshal(toolkitResponse)

				if toolErr != nil {
					fmt.Println("Toolkit handling error:", toolErr)
					toolResultBlock = anthropic.NewToolResultBlock(b.ID, string(toolResult), true)
				} else if marshalErr != nil {
					fmt.Println("Failed to marshal toolkit results:", marshalErr)
					toolResultBlock = anthropic.NewToolResultBlock(b.ID, fmt.Sprintf("Error marshaling result: %v", marshalErr), true)
				} else { // Success
					toolResultBlock = anthropic.NewToolResultBlock(b.ID, string(toolResult), false)
				}

				toolResultMessage := anthropic.MessageParam{
					Role: anthropic.F(anthropic.MessageParamRoleUser),
					Content: anthropic.F([]anthropic.ContentBlockParamUnion{
						toolResultBlock,
					}),
				}
				history = append(history, toolResultMessage)

			default:
				fmt.Println("Unexpected content block type:", b)
			}
		}

		if toolUsedInTurn {
			continue
		} else {
			finalThinking = thinkingInTurn
			finalMessageResponse = messageInTurn
			break
		}
	}
	return fmt.Sprintf("Thinking: %s\nMessage: %s", finalThinking, finalMessageResponse), nil
}

// GetToolkit returns the ToolParam definition for the orchestrator.
// It uses the locally initialized tkInstance.
func GetToolkit(tk *toolkit.Toolkit) anthropic.ToolParam {
	if tk == nil {
		log.Fatal("Error: Example toolkit (tkInstance) not initialized!") // Fatal in example
	}
	return anthropic.ToolParam{
		Name:        anthropic.F(tk.GetToolkitName()),
		Description: anthropic.F(tk.GetToolkitDescription()),
		InputSchema: anthropic.F(tk.GetToolkitSchema("anthropic")),
	}
}
