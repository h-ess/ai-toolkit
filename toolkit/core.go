// Package toolkit provides a hierarchical tool orchestration framework for AI-powered applications.
// It enables defining, composing, and executing multiple tools in a single invocation while
// handling schema generation, error management, and response aggregation automatically.
//
// Core concepts:
//   - Toolkit: The top-level container that manages multiple Parent tools
//   - Parent: A category of related tools that acts as a namespace for Child tools
//   - Child: An individual tool that performs a specific operation with args and returns results
//
// This file defines the core interfaces that all Parent and Child implementations must satisfy.
package toolkit

import (
	"context"
	"encoding/json"
)

// Parent represents a category of related tools (Children) that share a common purpose.
// It acts as a namespace and orchestrates the execution of its child tools.
// Implementations should handle execution errors gracefully for each child tool
// while preserving the structure of the response.
type Parent interface {
	// GetName returns the unique name of the parent toolset.
	// This name is used for lookup in toolkit requests and must be unique
	// within a toolkit instance.
	GetName() string

	// GetDescription provides a human-readable description of the parent's purpose.
	// This description is used in schema generation and documentation.
	GetDescription() string

	// GetChildren returns a map of the child tools managed by this parent,
	// keyed by their unique names. These names are used for lookup during execution.
	GetChildren() map[string]Child

	// HandleChildren processes a list of child tool execution requests.
	// It manages the execution workflow, including tool lookup, argument validation,
	// execution, and error handling, returning a consolidated ParentResponse.
	// The context should be propagated to each child's Handle method to support
	// cancellation, timeouts, and other context-aware operations.
	HandleChildren(ctx context.Context, childRequests []ToolKitChild) ParentResponse
}

// Child represents an individual tool or function that can be executed.
// Each Child belongs to a Parent and defines its own name, description,
// input schema, and execution logic. Implementations should handle their
// specific business logic while conforming to the expected interface.
type Child interface {
	// GetName returns the unique name of the child tool within its parent.
	// This name is used for lookup in execution requests and must be unique
	// within its parent.
	GetName() string

	// GetDescription provides a human-readable description of what the tool does.
	// This description is used in schema generation and documentation.
	GetDescription() string

	// GetInputSchema returns the JSON schema definition for the arguments
	// this tool expects. This schema is used for validation and documentation
	// in AI model prompts.
	GetInputSchema() interface{}

	// Handle executes the core logic of the tool with the provided arguments.
	// It unmarshals the raw JSON arguments into the expected type, performs
	// the operation, and returns the result or an error.
	// Implementations should use the provided context for cancellation support
	// and should return ToolKitError instances for structured error handling.
	Handle(ctx context.Context, args json.RawMessage) (interface{}, error)
}
