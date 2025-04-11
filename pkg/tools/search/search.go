package search

import (
	"context"
	"errors"
	"fmt"
	"log"
)

// --- Core Logic Functions (Exported) ---

// SearchWeb provides a simple mock implementation for web search.
// It returns fake results for demonstration purposes.
func SearchWeb(ctx context.Context, args SearchWebArgs) (SearchWebResponse, error) {
	log.Println("Executing Search Web with query:", args.Query)

	if args.Query == "" {
		return SearchWebResponse{
			Success: false,
			Error:   "query_required",
		}, errors.New("query_required")
	}

	// Return fake results
	return SearchWebResponse{
		Success: true,
		Results: []map[string]string{
			{"title": fmt.Sprintf("Example Result 1 for '%s'", args.Query), "url": "https://example.com/1"},
			{"title": fmt.Sprintf("Example Result 2 about '%s'", args.Query), "url": "https://example.com/2"},
		},
	}, nil
}

// FetchURLContent provides a simple mock implementation for fetching URL content.
// It returns fake HTML content for demonstration purposes.
func FetchURLContent(ctx context.Context, args FetchURLArgs) (FetchURLResponse, error) {
	log.Println("Executing Fetch URL Content for URL:", args.URL)

	if args.URL == "" {
		return FetchURLResponse{
			Success: false,
			Error:   "url_required",
		}, errors.New("url_required")
	}

	// Return fake HTML content
	fakeHTML := fmt.Sprintf("<html><body><h1>Mock Content for %s</h1><p>This is simulated content.</p></body></html>", args.URL)
	return FetchURLResponse{
		Success: true,
		Content: fakeHTML,
	}, nil

}
