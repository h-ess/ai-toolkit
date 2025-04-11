package search

// search specific types

// SearchWebArgs represents arguments for the SearchWeb operation
type SearchWebArgs struct {
	Query string `json:"query" jsonschema:"required,description=The search query string."`
}

// SearchWebResponse represents the response for the SearchWeb operation
type SearchWebResponse struct {
	Success bool        `json:"success"`
	Results interface{} `json:"results,omitempty"` // Use interface{} to allow flexibility, e.g., []string or more structured data
	Error   string      `json:"error,omitempty"`
}

// FetchURLArgs represents arguments for the FetchURLContent operation
type FetchURLArgs struct {
	URL string `json:"url" jsonschema:"required,description=The URL to fetch the content from."`
}

// FetchURLResponse represents the response for the FetchURLContent operation
type FetchURLResponse struct {
	Success bool   `json:"success"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}
