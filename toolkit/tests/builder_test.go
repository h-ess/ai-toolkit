package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	// Import the package we are testing
	// We use the exported functions like toolkit.NewChild
	"github.com/h-ess/ai-toolkit/toolkit"

	"github.com/stretchr/testify/assert"
)

// --- Test Structs ---

type SimpleArgs struct {
	Input string `json:"input" jsonschema:"required"`
}

type SimpleResponse struct {
	Output string `json:"output"`
}

// --- TestNewChild ---

func TestNewChild_Metadata(t *testing.T) {
	name := "test_child"
	desc := "A test child description"
	handler := func(ctx context.Context, args SimpleArgs) (interface{}, error) {
		return SimpleResponse{Output: "success:" + args.Input}, nil
	}

	child := toolkit.NewChild(name, desc, handler)

	if child.GetName() != name {
		t.Errorf("Expected name '%s', got '%s'", name, child.GetName())
	}
	if child.GetDescription() != desc {
		t.Errorf("Expected description '%s', got '%s'", desc, child.GetDescription())
	}

	schema := child.GetInputSchema()
	if schema == nil {
		t.Errorf("Expected input schema to be non-nil")
	}
	// Basic check: Schema is not nil. More detailed checks can be done
	// via GetToolkitDescription if necessary.
	assert.NotNil(t, schema)
}

func TestNewChild_Handle_Success(t *testing.T) {
	expectedOutput := "success:test_input"
	handler := func(ctx context.Context, args SimpleArgs) (interface{}, error) {
		return SimpleResponse{Output: expectedOutput}, nil
	}
	child := toolkit.NewChild("test_child_success", "desc", handler)

	inputArgsJSON := json.RawMessage(`{"input":"test_input"}`)
	ctx := context.Background()
	result, err := child.Handle(ctx, inputArgsJSON)

	if err != nil {
		t.Fatalf("Handle failed unexpectedly: %v", err)
	}

	resp, ok := result.(SimpleResponse)
	if !ok {
		t.Fatalf("Expected result type SimpleResponse, got %T", result)
	}
	if resp.Output != expectedOutput {
		t.Errorf("Expected output '%s', got '%s'", expectedOutput, resp.Output)
	}
}

func TestNewChild_Handle_HandlerError(t *testing.T) {
	expectedError := errors.New("handler failed")
	handler := func(ctx context.Context, args SimpleArgs) (interface{}, error) {
		return nil, expectedError
	}
	child := toolkit.NewChild("test_child_handler_err", "desc", handler)

	inputArgsJSON := json.RawMessage(`{"input":"test"}`)
	ctx := context.Background()
	_, err := child.Handle(ctx, inputArgsJSON)

	if err == nil {
		t.Fatal("Expected an error from Handle, got nil")
	}

	// Check if it's the expected type of toolkit error
	tkErr, ok := err.(toolkit.ToolKitError)
	if !ok {
		t.Fatalf("Expected error type toolkit.ToolKitError, got %T", err)
	}
	if tkErr.Code != "handler_execution_error" {
		t.Errorf("Expected error code 'handler_execution_error', got '%s'", tkErr.Code)
	}
	// We could also check if the message contains the original error string
	t.Logf("Got expected error: %v", err) // Log for confirmation
}

func TestNewChild_Handle_UnmarshalError(t *testing.T) {
	handler := func(ctx context.Context, args SimpleArgs) (interface{}, error) {
		// This shouldn't be called
		t.Fatal("Handler called unexpectedly on unmarshal error")
		return nil, nil
	}
	child := toolkit.NewChild("test_child_unmarshal_err", "desc", handler)

	inputArgsJSON := json.RawMessage(`{"bad`)
	ctx := context.Background()
	_, err := child.Handle(ctx, inputArgsJSON)

	if err == nil {
		t.Fatal("Expected an error from Handle, got nil")
	}

	tkErr, ok := err.(toolkit.ToolKitError)
	if !ok {
		t.Fatalf("Expected error type toolkit.ToolKitError, got %T", err)
	}
	if tkErr.Code != "invalid_arguments" {
		t.Errorf("Expected error code 'invalid_arguments', got '%s'", tkErr.Code)
	}
	t.Logf("Got expected error: %v", err) // Log for confirmation
}

// --- TestNewParent ---

// Helper function to create a simple child for parent tests
func createTestChild(t *testing.T, name string, output string, shouldError bool) toolkit.Child {
	t.Helper()
	handler := func(ctx context.Context, args SimpleArgs) (interface{}, error) {
		if shouldError {
			return nil, fmt.Errorf("error_from_%s", name)
		}
		return SimpleResponse{Output: fmt.Sprintf("%s:%s", output, args.Input)}, nil
	}
	return toolkit.NewChild[SimpleArgs](name, "desc_"+name, handler)
}

func TestNewParent_Metadata(t *testing.T) {
	name := "test_parent"
	desc := "A test parent description"
	child1 := createTestChild(t, "child1", "out1", false)

	parent := toolkit.NewParent(name, desc, child1)

	if parent.GetName() != name {
		t.Errorf("Expected parent name '%s', got '%s'", name, parent.GetName())
	}
	if parent.GetDescription() != desc {
		t.Errorf("Expected parent description '%s', got '%s'", desc, parent.GetDescription())
	}
}

func TestNewParent_GetChildren(t *testing.T) {
	child1 := createTestChild(t, "child1", "out1", false)
	child2 := createTestChild(t, "child2", "out2", false)

	parent := toolkit.NewParent("test_parent_get", "desc", child1, child2)

	childrenMap := parent.GetChildren()
	if len(childrenMap) != 2 {
		t.Fatalf("Expected 2 children, got %d", len(childrenMap))
	}

	if _, ok := childrenMap["child1"]; !ok {
		t.Error("Expected child 'child1' to be in the map")
	}
	if _, ok := childrenMap["child2"]; !ok {
		t.Error("Expected child 'child2' to be in the map")
	}
	if _, ok := childrenMap["child3"]; ok {
		t.Error("Did not expect child 'child3' to be in the map")
	}
}

func TestNewParent_HandleChildren_Success(t *testing.T) {
	child1 := createTestChild(t, "child1", "out1", false)
	child2 := createTestChild(t, "child2", "out2", false)
	parent := toolkit.NewParent("test_parent_success", "desc", child1, child2)

	requests := []toolkit.ToolKitChild{
		{Name: "child1", Args: json.RawMessage(`{"input":"in1"}`)},
		{Name: "child2", Args: json.RawMessage(`{"input":"in2"}`)},
	}

	ctx := context.Background()
	parentResp := parent.HandleChildren(ctx, requests)

	if parentResp.Name != "test_parent_success" {
		t.Errorf("Unexpected parent response name: %s", parentResp.Name)
	}
	if len(parentResp.ChildsResponses) != 2 {
		t.Fatalf("Expected 2 child responses, got %d", len(parentResp.ChildsResponses))
	}

	// Check response 1
	resp1 := parentResp.ChildsResponses[0]
	if resp1.Name != "child1" {
		t.Errorf("Expected child name 'child1', got '%s'", resp1.Name)
	}
	result1, ok := resp1.Response.(SimpleResponse)
	if !ok {
		t.Fatalf("Expected response type SimpleResponse for child1, got %T", resp1.Response)
	}
	if result1.Output != "out1:in1" {
		t.Errorf("Expected output 'out1:in1', got '%s'", result1.Output)
	}

	// Check response 2
	resp2 := parentResp.ChildsResponses[1]
	if resp2.Name != "child2" {
		t.Errorf("Expected child name 'child2', got '%s'", resp2.Name)
	}
	result2, ok := resp2.Response.(SimpleResponse)
	if !ok {
		t.Fatalf("Expected response type SimpleResponse for child2, got %T", resp2.Response)
	}
	if result2.Output != "out2:in2" {
		t.Errorf("Expected output 'out2:in2', got '%s'", result2.Output)
	}
}

func TestNewParent_HandleChildren_ChildNotFound(t *testing.T) {
	child1 := createTestChild(t, "child1", "out1", false)
	parent := toolkit.NewParent("test_parent_notfound", "desc", child1)

	requests := []toolkit.ToolKitChild{
		{Name: "child1", Args: json.RawMessage(`{"input":"in1"}`)},
		{Name: "non_existent_child", Args: json.RawMessage(`{}`)},
	}

	ctx := context.Background()
	parentResp := parent.HandleChildren(ctx, requests)

	if len(parentResp.ChildsResponses) != 2 {
		t.Fatalf("Expected 2 child responses, got %d", len(parentResp.ChildsResponses))
	}

	// Check response for non-existent child
	resp2 := parentResp.ChildsResponses[1]
	if resp2.Name != "non_existent_child" {
		t.Errorf("Expected child name 'non_existent_child', got '%s'", resp2.Name)
	}
	tkErr, ok := resp2.Response.(toolkit.ToolKitError)
	if !ok {
		t.Fatalf("Expected response type toolkit.ToolKitError for non_existent_child, got %T", resp2.Response)
	}
	if tkErr.Code != "child_not_found" {
		t.Errorf("Expected error code 'child_not_found', got '%s'", tkErr.Code)
	}
	t.Logf("Got expected error for non-existent child: %v", tkErr)
}

func TestNewParent_HandleChildren_ChildError(t *testing.T) {
	child1 := createTestChild(t, "child1", "out1", false)            // Should succeed
	childWithError := createTestChild(t, "childWithError", "", true) // Should error
	parent := toolkit.NewParent("test_parent_child_err", "desc", child1, childWithError)

	requests := []toolkit.ToolKitChild{
		{Name: "child1", Args: json.RawMessage(`{"input":"in1"}`)},
		{Name: "childWithError", Args: json.RawMessage(`{"input":"inErr"}`)},
	}

	ctx := context.Background()
	parentResp := parent.HandleChildren(ctx, requests)

	if len(parentResp.ChildsResponses) != 2 {
		t.Fatalf("Expected 2 child responses, got %d", len(parentResp.ChildsResponses))
	}

	// Check response for childWithError
	resp2 := parentResp.ChildsResponses[1]
	if resp2.Name != "childWithError" {
		t.Errorf("Expected child name 'childWithError', got '%s'", resp2.Name)
	}
	tkErr, ok := resp2.Response.(toolkit.ToolKitError)
	if !ok {
		t.Fatalf("Expected response type toolkit.ToolKitError for childWithError, got %T", resp2.Response)
	}
	if tkErr.Code != "handler_execution_error" {
		t.Errorf("Expected error code 'handler_execution_error', got '%s'", tkErr.Code)
	}
	t.Logf("Got expected error from child handler: %v", tkErr)
}
