# AI-Toolkit

A high-performance hierarchical tool orchestration framework for AI-powered applications.

## Problem Statement

Traditional AI tool usage faces a critical limitation: **one tool per invocation per turn**.

This creates several challenges:
- **High Latency**: Each tool call requires a full round-trip (AI model → backend → tool execution → backend → AI model)
- **Chatty Interfaces**: Complex workflows need multiple exchanges between model and backend
- **Inefficient Token Usage**: Repeated context sending across multiple turns
- **Complex State Management**: Application code must track state across multiple tool invocations

## Solution

AI-Toolkit enables **multiple tool invocations in a single turn** through a hierarchical orchestration framework:

```
┌─────────────────────────────────┐
│ Single API Call                 │
│                                 │
│  ┌─────────────┐ ┌─────────────┐│
│  │ Parent: DB  │ │ Parent: FS  ││
│  │ ┌─────────┐ │ │ ┌─────────┐ ││
│  │ │ Query   │ │ │ │ ReadFile│ ││
│  │ └─────────┘ │ │ └─────────┘ ││
│  │ ┌─────────┐ │ │ ┌─────────┐ ││
│  │ │ Insert  │ │ │ │ WriteFile││
│  │ └─────────┘ │ │ └─────────┘ ││
│  └─────────────┘ └─────────────┘│
└─────────────────────────────────┘
```

**Benefits:**
- **5-10x Latency Reduction**: Single API call vs multiple sequential calls
- **Structured Organization**: Logical grouping of related tools
- **Automatic Schema Generation**: JSON schemas derived from Go types
- **Clean Error Handling**: Standardized error propagation and collection
- **Parallel Execution Capability**: Multiple tools execute in a single turn

## Architecture

AI-Toolkit is built around three core concepts:

### 1. Toolkit
The top-level orchestrator that manages Parent categories and handles request routing.

### 2. Parent
A category of related tools that acts as a namespace and container.

### 3. Child
Individual tool implementations that perform specific operations.

```
┌───────────────────────────────────────┐
│ Toolkit                               │
│ ┌───────────────┐  ┌────────────────┐ │
│ │ Parent: Files │  │ Parent: Search │ │
│ │ ┌───────────┐ │  │ ┌────────────┐ │ │
│ │ │ ReadFile  │ │  │ │ WebSearch  │ │ │
│ │ └───────────┘ │  │ └────────────┘ │ │
│ │ ┌───────────┐ │  │ ┌────────────┐ │ │
│ │ │ WriteFile │ │  │ │ FetchURL   │ │ │
│ │ └───────────┘ │  │ └────────────┘ │ │
│ └───────────────┘  └────────────────┘ │
└───────────────────────────────────────┘
```

## Data Flow

```
┌──────────┐     ┌─────────┐     ┌──────────┐     ┌──────────┐
│ AI Model │────▶│ Toolkit │────▶│ Parent A │────▶│ Child A1 │
│          │     │         │     │          │     └──────────┘
│          │     │         │     │          │     ┌──────────┐
│          │     │         │     │          │────▶│ Child A2 │
│          │     │         │     └──────────┘     └──────────┘
│          │     │         │     ┌──────────┐     ┌──────────┐
│          │     │         │────▶│ Parent B │────▶│ Child B1 │
└──────────┘     └─────────┘     └──────────┘     └──────────┘
```

## Installation

```bash
go get github.com/hamzaessahbaoui/ai-toolkit
```

## Quick Start

### 1. Define Your Tools

```go
package main

import (
    "context"
    "fmt"
    "os"
    
    "github.com/your-username/ai-toolkit/toolkit"
)

// Define argument and response types
type ReadFileArgs struct {
    Path string `json:"path" jsonschema:"required,description=The path to the file"`
}

type ReadFileResponse struct {
    Content string `json:"content"`
    Success bool   `json:"success"`
}

func main() {
    // Create a child tool
    readFileTool := toolkit.NewChild(
        "read_file",
        "Reads the content of a file",
        func(ctx context.Context, args ReadFileArgs) (interface{}, error) {
            content, err := os.ReadFile(args.Path)
            if err != nil {
                return ReadFileResponse{Success: false}, err
            }
            return ReadFileResponse{
                Content: string(content),
                Success: true,
            }, nil
        },
    )
    
    // Create a parent to group related tools
    fileOpsParent := toolkit.NewParent(
        "file_operations",
        "File system operations",
        readFileTool,
        // Add more child tools...
    )
    
    // Create the toolkit
    myToolkit := toolkit.New(
        "my_app_toolkit",
        fileOpsParent,
        // Add more parents...
    )
    
    // The toolkit is now ready to handle requests
}
```

### 2. Process Tool Requests

```go
// Process a JSON request from an AI model
func handleToolRequest(toolkitJSON []byte) {
    ctx := context.Background()
    
    // Process the toolkit request
    response, err := myToolkit.HandleToolKit(ctx, toolkitJSON)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    // Handle the response
    fmt.Printf("Response: %+v\n", response)
}
```

## Real-World Example with Claude

For a complete working example, see the [Claude integration example](examples/claude/main.go) in this repository.

```go
import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    
    "github.com/anthropics/anthropic-sdk-go"
    "github.com/anthropics/anthropic-sdk-go/option"
    
    "github.com/your-username/ai-toolkit/toolkit"
)

// Create your toolkit with various tools...

// Configure Claude with the toolkit
params := anthropic.MessageNewParams{
    Model: anthropic.F(anthropic.ModelClaude3_7Sonnet20250219),
    System: anthropic.F([]anthropic.TextBlockParam{
        anthropic.NewTextBlock(
            `You are a helpful assistant. You can execute multiple tools in one invocation.`),
    }),
    Tools: anthropic.F([]anthropic.ToolUnionUnionParam{
        anthropic.ToolParam{
            Name:        anthropic.F(myToolkit.GetToolkitName()),
            Description: anthropic.F(myToolkit.GetToolkitDescription()),
            InputSchema: anthropic.F(myToolkit.GetToolkitSchema("anthropic")),
        },
    }),
}

// Later, in your conversation loop, handle tool use:

for _, block := range claudeResponse.Content {
    switch b := block.AsUnion().(type) {
    case anthropic.ToolUseBlock:
        // Handle the tool use request
        toolkitResponse, err := myToolkit.HandleToolKit(ctx, b.Input)
        
        // Send the result back to Claude
        toolResultJSON, _ := json.Marshal(toolkitResponse)
        toolResultBlock := anthropic.NewToolResultBlock(b.ID, string(toolResultJSON), err != nil)
        
        // Add result to conversation history...
    }
}
```

## Use Cases

AI-Toolkit excels in scenarios requiring complex, multi-step tool workflows:

1. **Knowledge Work**: Query multiple data sources, filter results, generate summaries
2. **Content Creation**: Research topics, fetch references, generate and publish content
3. **Data Analysis**: Extract data, transform it, analyze patterns, visualize results
4. **Process Automation**: Create multi-step workflows with conditional branches and parallel execution

## Technical Design

### Builder Pattern

AI-Toolkit uses a builder pattern with Go generics to create type-safe tools:

```go
// Create a strongly-typed tool with automatic schema generation
toolkit.NewChild[MyArgType](
    "tool_name",
    "Tool description",
    func(ctx context.Context, args MyArgType) (interface{}, error) {
        // Implement tool logic here
        return result, nil
    },
)
```

### Context Propagation

All toolkit operations accept and propagate `context.Context` for cancellation support, timeouts, and value passing:

```go
// Context flows from request handling down to individual tools
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// The toolkit propagates this context to all tools
response, err := myToolkit.HandleToolKit(ctx, requestJSON)
```

### JSON Schema Generation

The toolkit automatically generates JSON schemas from Go types using struct tags:

```go
type UserArgs struct {
    Name     string `json:"name" jsonschema:"required,description=The user's name"`
    Age      int    `json:"age" jsonschema:"description=The user's age in years"`
    Location string `json:"location" jsonschema:"description=The user's location"`
}
```

### Error Handling

Standardized error handling with structured error types:

```go
// Return structured errors from tools
if err != nil {
    return nil, toolkit.NewError("file_not_found", fmt.Sprintf("File %s not found", path))
}

// Error responses are included in the response structure
response, err := myToolkit.HandleToolKit(ctx, requestJSON)
// Even if err != nil, response contains structured error information
```

## Comparison with Traditional Approach

| Feature | Traditional Approach | AI-Toolkit |
|---------|---------------------|------------|
| Latency | High (multiple round trips) | Low (single round trip) |
| Complexity | Complex state management | Simple hierarchical structure |
| Extensibility | Ad-hoc | Structured parent/child system |
| Error Handling | Inconsistent | Standardized |
| Schema Generation | Manual | Automatic from Go types |

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
