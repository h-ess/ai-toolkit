package operations

// EditFileArgs represents arguments for the EditFile operation
type EditFileArgs struct {
	Path    string `json:"path" jsonschema:"required,description=The absolute or relative path to the file to write."`
	Content string `json:"content" jsonschema:"required,description=The content to write into the file."`
}

// EditFileResponse represents the response for the EditFile operation
type EditFileResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"` // Only present on failure
}

// ReadFileArgs represents arguments for the ReadFile operation
type ReadFileArgs struct {
	Path string `json:"path" jsonschema:"required,description=The absolute or relative path to the file to read."`
}

// ReadFileResponse represents the response for the ReadFile operation
type ReadFileResponse struct {
	Success bool   `json:"success"`
	Content string `json:"content,omitempty"` // Only present on success
	Error   string `json:"error,omitempty"`   // Only present on failure
}

// ReadFileInfo and EditFileInfo variables removed as they are no longer needed.
// Schema is generated dynamically by the toolkit.NewChild builder.
