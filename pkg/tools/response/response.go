package response

import (
	"context"
	"log"
)

// --- Core Logic Functions (Now Exported) ---

// AIThinking performs the actual logging.
// Renamed to be exported.
func LogThinking(ctx context.Context, args ModelThinkingArgs) (ModelThinking, error) {
	log.Println("Executing Model Thinking")
	// Context is available but unused in this simple logging function
	if args.Thinking == "" {
		log.Println("Model Thinking Warning: Received empty thinking string.")
		return ModelThinking{Success: true}, nil
	}
	log.Printf("--- MODEL THINKING START ---\n%s\n--- MODEL THINKING END ---", args.Thinking)
	return ModelThinking{
		Success: true,
	}, nil
}

// LogResponse logs that a response call was made.
func LogResponse(ctx context.Context, args ModelResponseArgs) (ModelResponse, error) {
	log.Println("Executing Model Response.")
	if args.Response == "" {
		log.Println("Model Response Warning: Received empty user response string.")
		return ModelResponse{Success: true}, nil
	}
	log.Printf("--- MODEL RESPONSE START ---\n%s\n--- MODEL RESPONSE END ---", args.Response)
	return ModelResponse{Success: true}, nil
}
