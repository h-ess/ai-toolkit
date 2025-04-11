package operations

import (
	"context"
	"errors"
	"log"
	"os"
)

// --- Core Logic Functions (Now Exported) ---

// EditFile performs the actual file writing.
// Renamed to be exported.
func EditFile(ctx context.Context, args EditFileArgs) (EditFileResponse, error) {
	log.Println("Execute Edit File, with: ", args)

	if args.Path == "" {
		return EditFileResponse{
			Success: false,
			Error:   "path_required",
		}, errors.New("path_required")
	}

	err := os.WriteFile(args.Path, []byte(args.Content), 0644)
	if err != nil {
		log.Printf("Execute Edit File - Error: Failed to write file %s: %v", args.Path, err)
		return EditFileResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return EditFileResponse{
		Success: true,
	}, nil
}

// ReadFile performs the actual file reading.
// Renamed to be exported.
func ReadFile(ctx context.Context, args ReadFileArgs) (ReadFileResponse, error) {
	log.Println("Execute Read File, with: ", args)

	if args.Path == "" {
		return ReadFileResponse{
			Success: false,
			Error:   "path_required",
		}, errors.New("path_required")
	}

	content, err := os.ReadFile(args.Path)
	if err != nil {
		log.Printf("Execute Read File - Error: Failed to read file %s: %v", args.Path, err)
		return ReadFileResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return ReadFileResponse{
		Success: true,
		Content: string(content),
	}, nil
}

// --- Builder Handler Functions (Removed) ---
// Wrapper functions like handleEditFile are removed.
// The consumer (e.g., internal/claude/tools.go) will define these.

// --- Parent Creation (Removed) ---
// CreateFileOperationsParent function is removed.
// Parent creation is now the responsibility of the consumer.
