package response

// Removed unused import
// import "toolkit/pkg/toolkit/types"

// --- Predefined Child Tool Information ---
// Removed LogThinkingInfo and LogResponseInfo variables.
// Schema generation handled by toolkit.NewChild.

// --- Argument Structs for Child Tools ---

// ModelThinkingArgs defines the arguments for the model_thinking tool.
type ModelThinkingArgs struct {
	Thinking string `json:"thinking" jsonschema:"required,description=The thinking steps or thoughts to be logged by the system."`
}

// ModelResponseArgs defines the arguments for the model_response tool.
type ModelResponseArgs struct {
	Response string `json:"response" jsonschema:"required,description=The final response text to be presented to the user."`
}

// --- Response Structs for Child Tools ---

// ModelThinking defines the success response for the model_thinking tool.
type ModelThinking struct {
	Success bool   `json:"Success"`
	Error   string `json:"Error,omitempty"` // Only present on failure
}

// ModelResponse defines the success response for the model_response tool.
type ModelResponse struct {
	Success bool   `json:"Success"`
	Error   string `json:"Error,omitempty"` // Only present on failure
}
