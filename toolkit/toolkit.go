// Package toolkit provides a hierarchical tool orchestration framework for AI-powered applications.
// It enables defining, composing, and executing multiple tools in a single invocation while
// handling schema generation, error management, and response aggregation automatically.
package toolkit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// --- Toolkit Struct and Methods ---

// Toolkit orchestrates the execution of hierarchical tool structures.
// It serves as the top-level container for Parent tools and provides methods
// for generating descriptions, JSON schemas, and processing execution requests.
// Each Toolkit instance maintains a registry of Parent tools identified by unique names.
type Toolkit struct {
	parents map[string]Parent // Registry of Parent implementations mapped by name
	name    string            // Name of this toolkit instance
}

// New creates a new Toolkit instance with the provided name and parent toolkits.
// It registers the provided Parent implementations for execution and provides
// safeguards against nil or duplicate parents.
//
// Parameters:
//   - name: A unique identifier for this toolkit instance
//   - parents: A variadic list of Parent implementations to register in this toolkit
//
// Behavior:
//   - Nil parents are skipped with a warning
//   - If duplicate parent names are detected, the last one overwrites previous instances
//   - No default parents are added automatically
//
// Returns:
//   - A pointer to the initialized Toolkit instance
//
// Example:
//
//	fileOpsParent := toolkit.NewParent("file_ops", "File operations", fileReadTool, fileWriteTool)
//	networkParent := toolkit.NewParent("network", "Network operations", httpFetchTool)
//	toolkit := toolkit.New("my_toolkit", fileOpsParent, networkParent)
func New(name string, parents ...Parent) *Toolkit {
	parentMap := make(map[string]Parent, len(parents))
	for _, p := range parents {
		if p == nil {
			log.Println("Warning: nil parent provided to toolkit.New, skipping.")
			continue
		}
		if _, exists := parentMap[p.GetName()]; exists {
			log.Printf("Warning: Duplicate parent name '%s' detected in toolkit.New. Overwriting.", p.GetName())
		}
		parentMap[p.GetName()] = p
	}

	return &Toolkit{
		parents: parentMap,
		name:    name,
	}
}

// GetToolkitName returns the configured name of the toolkit instance.
// This name is used in responses and can be useful for identifying which toolkit
// handled a particular request in multi-toolkit environments.
func (t *Toolkit) GetToolkitName() string {
	return t.name
}

// GetToolkitSchema returns a JSON schema representation for the toolkit's request structure.
// The schema is provider-specific and currently supports "anthropic" (Claude) format,
// which is used as the default for unsupported providers.
//
// Parameters:
//   - provider: The target provider identifier (e.g., "anthropic" for Claude)
//
// Returns:
//   - A JSON schema object suitable for the specified provider
//
// The schema includes the full structure of the ToolKit request format, including
// definitions for parents and children, and is suitable for direct use with LLM
// tool registration endpoints.
func (t *Toolkit) GetToolkitSchema(provider string) interface{} {
	switch provider {
	case "anthropic":
		return GetToolKitSchemaForAnthropic()
	default:
		log.Printf("Warning: Unsupported schema provider '%s', defaulting to Anthropic schema", provider)
		return GetToolKitSchemaForAnthropic()
	}
}

// GetToolkitDescription generates a human-readable XML-like description of the toolkit structure.
// This description is typically used for providing context to language models about
// the available tools and their capabilities.
//
// The description includes:
//   - The toolkit name and a general explanation of the toolkit structure
//   - A list of all parents with their descriptions
//   - For each parent, a list of its child tools with descriptions and input schemas
//
// Returns:
//   - A formatted string containing the full toolkit description
//
// This description is designed to be understood by LLMs for effective tool use
// and follows a consistent XML-like format that highlights the hierarchical structure.
func (t *Toolkit) GetToolkitDescription() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("In this environment, you have access to the following <toolkit name=\"%s\">:\n", t.name))
	sb.WriteString("A <toolkit> is a collection of <parents>, a <parent> is a collection of <childs>.\n")
	sb.WriteString("Below is the list of available <parents> and their <childs>:\n")

	for _, parent := range t.parents {
		sb.WriteString(fmt.Sprintf("<parent name=\"%s\" description=\"%s\"></parent>\n", parent.GetName(), parent.GetDescription()))

		children := parent.GetChildren()
		if len(children) > 0 {
			// TODO: Maybe sort children by name?
			for _, child := range children {
				schema := child.GetInputSchema()
				schemaBytes, err := json.Marshal(schema)
				schemaStr := "schema_error"
				if err == nil {
					schemaStr = string(schemaBytes)
				} else {
					log.Printf("Error marshaling schema for %s.%s: %v", parent.GetName(), child.GetName(), err)
				}
				sb.WriteString(fmt.Sprintf("<child name=\"%s\" description=\"%s\"><input_schema>%s</input_schema></child>\n", child.GetName(), child.GetDescription(), schemaStr))
			}
			sb.WriteString("</parent>\n")
		} else {
			sb.WriteString("</parent>\n")
		}
		sb.WriteString("**NOTE**: A child tool cannot be invoked directly, the parent tool must be invoked first via its parent.\n")
	}
	sb.WriteString("</toolkit>")

	return sb.String()
}

// --- Processing Methods ---

// HandleToolKit is the main entry point for processing toolkit execution requests.
// It accepts raw JSON input containing parent and child tool invocations, parses it,
// and orchestrates the execution of the requested tools.
//
// Parameters:
//   - ctx: The execution context for propagating cancellation, deadlines, and values
//   - input: Raw JSON payload containing the toolkit request (must follow ToolKit structure)
//
// Returns:
//   - ToolKitResponse: Hierarchical response containing results from all executed tools
//   - error: Any error encountered during processing, or nil on success
//
// The method handles various failure scenarios gracefully:
//   - If JSON parsing fails, it returns a structured error response
//   - If requested parents are not found, it includes error responses for those parents
//   - If child tools fail, their errors are included in the appropriate child responses
//
// This enables clients to process both successful and failed operations in a consistent way.
func (t *Toolkit) HandleToolKit(ctx context.Context, input json.RawMessage) (ToolKitResponse, error) {
	tkRequest, err := t.parseToolKitInput(input)
	if err != nil {
		// Return a structured error response for parsing errors
		log.Printf("Error parsing toolkit input: %v", err)
		errResp := ToolKitResponse{
			Name: "toolkit_request_parse_error",
			Responses: []ParentResponse{
				{
					Name: "_parse_error",
					ChildsResponses: []ChildResponse{
						{Name: "_input_error", Response: NewError("invalid_input_json", err.Error())},
					},
				},
			},
		}
		return errResp, err
	}

	// Pass the parsed request and context to the internal toolkit processor
	return t.processToolKit(ctx, tkRequest)
}

// processToolKit orchestrates the execution of tools based on a parsed request.
// It routes each parent request to the appropriate Parent instance and collects
// their responses into a unified structure.
//
// This is an internal method used by HandleToolKit and shouldn't be called directly.
func (t *Toolkit) processToolKit(ctx context.Context, toolkitRequest ToolKit) (ToolKitResponse, error) {
	tlResponse := ToolKitResponse{
		Name: t.GetToolkitName(),
	}

	if len(toolkitRequest.ToolKitParents) == 0 {
		return tlResponse, NewError("no_toolkit_parents", "No toolkit parents specified in the request")
	}

	for _, parentReq := range toolkitRequest.ToolKitParents {
		parent, ok := t.parents[parentReq.Name]
		if !ok {
			log.Printf("Toolkit: Requested parent '%s' not found", parentReq.Name)
			errResp := ParentResponse{
				Name: parentReq.Name,
				ChildsResponses: []ChildResponse{
					{Name: "_parent_error", Response: NewError("parent_not_found", fmt.Sprintf("Parent toolkit '%s' not registered", parentReq.Name))},
				},
			}
			tlResponse.AddResponse(errResp)
			continue
		}

		// Pass context down to HandleChildren
		parentResponse := parent.HandleChildren(ctx, parentReq.ToolKitChilds)
		tlResponse.AddResponse(parentResponse)
	}

	return tlResponse, nil
}

// parseToolKitInput parses the incoming JSON request into a structured format.
// It validates that the JSON conforms to the expected ToolKit structure.
//
// This is an internal method used by HandleToolKit and shouldn't be called directly.
func (t *Toolkit) parseToolKitInput(input json.RawMessage) (ToolKit, error) {
	var toolkitRequest ToolKit
	if err := json.Unmarshal(input, &toolkitRequest); err != nil {
		return ToolKit{}, fmt.Errorf("error unmarshaling toolkit JSON input: %w", err)
	}
	return toolkitRequest, nil
}
