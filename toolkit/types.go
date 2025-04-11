// Package toolkit provides a hierarchical tool orchestration framework for AI-powered applications.
// This file defines the data structures used for requests, responses, errors,
// and schema generation within the toolkit framework.
package toolkit

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
)

// --- Core Toolkit Request/Response Structures ---

// ToolKit represents the top-level structure of a toolkit execution request.
// It defines the format that AI models or external clients use to invoke tools.
// Multiple parent and child tools can be invoked in a single request, enabling
// parallel execution and reducing round-trip latency.
type ToolKit struct {
	Name           string          `json:"name" jsonschema:"required,description=The name of the toolkit."`
	ToolKitParents []ToolKitParent `json:"parents" jsonschema:"required,description=The parent toolkits to execute within the toolkit."`
}

// ToolKitParent represents a specific parent toolkit requested for execution within a ToolKit request.
// It encapsulates a collection of related child tools that should be executed together under
// the same parent namespace.
type ToolKitParent struct {
	Name          string         `json:"name" jsonschema:"required,description=The name of the parent toolkit to execute."`
	ToolKitChilds []ToolKitChild `json:"childs" jsonschema:"required,description=The child tools to execute within this parent."`
}

// ToolKitChild represents an individual child tool requested for execution within a ToolKitParent request.
// It holds the tool name and its arguments as raw JSON, allowing for delayed parsing by the specific
// tool handler during execution.
type ToolKitChild struct {
	Name string          `json:"name" jsonschema:"required,description=The name of the child tool to execute."`
	Args json.RawMessage `json:"args" jsonschema:"required,description=The arguments for the child tool, as a JSON object."`
}

// ToolKitResponse represents the top-level structure of the response returned after processing a ToolKit request.
// It preserves the hierarchical structure of the request, containing responses from all parent and child
// tools that were executed.
type ToolKitResponse struct {
	Name      string           `json:"name"`
	Responses []ParentResponse `json:"responses,omitempty"`
}

// ParentResponse represents the aggregated response from processing a specific parent toolkit within a request.
// It contains the parent's name and an ordered list of responses from each child tool that was executed,
// maintaining the original execution order.
type ParentResponse struct {
	Name            string          `json:"name"`
	ChildsResponses []ChildResponse `json:"childsResponses,omitempty"`
}

// ChildResponse represents the response from executing a single child tool.
// The Response field can contain either the successful result (as returned by the tool's handler)
// or a ToolKitError if an error occurred during execution, providing a consistent error handling mechanism.
type ChildResponse struct {
	Name     string      `json:"name"`
	Response interface{} `json:"response,omitempty"`
}

// --- Tool Metadata Structures ---

// ChildInfo struct is removed.

// --- Error Handling ---

// ToolKitError provides a standardized structure for errors occurring within the toolkit framework.
// It encapsulates both a machine-readable error code for programmatic handling and a human-readable
// message for debugging and user feedback.
type ToolKitError struct {
	Code    string `json:"Code"`    // A machine-readable error code (e.g., "invalid_arguments", "handler_execution_error")
	Message string `json:"Message"` // A human-readable description of the error
}

// Error implements the standard error interface for ToolKitError.
// This enables ToolKitError to be used with standard Go error handling mechanisms
// while preserving the structured error information.
func (e ToolKitError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewError creates a new ToolKitError instance with the specified code and message.
// This is the preferred way to create and return errors from tool implementations
// to ensure consistent error handling across the toolkit.
//
// Common error codes:
//   - "invalid_arguments": When tool arguments don't match the expected schema
//   - "handler_execution_error": When the tool execution fails
//   - "child_not_found": When a requested child tool doesn't exist
//   - "parent_not_found": When a requested parent doesn't exist
func NewError(code, message string) error {
	return ToolKitError{
		Code:    code,
		Message: message,
	}
}

// --- Response Helper Methods ---

// AddResponse appends a ParentResponse to the ToolKitResponse's list of responses.
// This helper method is used during the toolkit processing workflow to build
// the hierarchical response structure.
func (tr *ToolKitResponse) AddResponse(pr ParentResponse) {
	tr.Responses = append(tr.Responses, pr)
}

// AddResponse appends a ChildResponse to the ParentResponse's list of child responses.
// This helper method is used by Parent implementations to build the response
// structure during child tool execution.
func (pr *ParentResponse) AddResponse(cr ChildResponse) {
	pr.ChildsResponses = append(pr.ChildsResponses, cr)
}

// --- Schema Generation Helper ---

// GenerateSchema creates a JSON schema representation for the provided generic type T.
// It uses reflection through the github.com/invopop/jsonschema library to generate
// a complete schema that can be used for documentation, validation, and providing
// to LLMs for tool use.
//
// The schema generation respects jsonschema tags on struct fields, including:
// - required: Whether the field is required
// - description: Field descriptions for documentation
//
// Example usage:
//
//	type MyArgs struct {
//	    Name string `json:"name" jsonschema:"required,description=The user's name"`
//	    Age  int    `json:"age" jsonschema:"description=The user's age in years"`
//	}
//	schema := GenerateSchema[MyArgs]()
func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties:  true, // Allow additional properties in the generated schema
		DoNotReference:             true, // Keep schema self-contained, no $refs
		RequiredFromJSONSchemaTags: true, // Respect `jsonschema:"required"` tags
	}
	var v T                      // Create a zero value instance of the type T
	return reflector.Reflect(&v) // Reflect on the zero value to get the schema
}

// GetToolKitSchemaForAnthropic generates the specific JSON schema structure
// expected by Anthropic's Claude API for the top-level ToolKit request.
// This schema is used when registering the toolkit with Anthropic's tool use capability.
func GetToolKitSchemaForAnthropic() interface{} {
	return GenerateSchema[ToolKit]()
}
