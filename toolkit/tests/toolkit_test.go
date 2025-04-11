package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/h-ess/ai-toolkit/toolkit"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test Helpers (Redefined for simplicity) ---

type testArgs struct {
	Val string `json:"val"`
}

type testResp struct {
	Res string `json:"res"`
}

func createTestParent(t *testing.T, name string, children ...toolkit.Child) toolkit.Parent {
	t.Helper()
	return toolkit.NewParent(name, "desc_"+name, children...)
}

func createTestChildFn(t *testing.T, name string, retVal string, shouldErr bool) toolkit.Child {
	t.Helper()
	handler := func(ctx context.Context, args testArgs) (interface{}, error) {
		if shouldErr {
			return nil, fmt.Errorf("child_err_%s", name)
		}
		return testResp{Res: retVal + ":" + args.Val}, nil
	}
	return toolkit.NewChild[testArgs](name, "desc_"+name, handler)
}

// --- Test New ---

func TestNew(t *testing.T) {
	parent1 := createTestParent(t, "parent1", createTestChildFn(t, "child1a", "res1a", false))
	parent2 := createTestParent(t, "parent2", createTestChildFn(t, "child2a", "res2a", false))

	tests := []struct {
		name        string
		kName       string
		parents     []toolkit.Parent
		expectCount int
		expectNames []string
	}{
		{
			name:        "no parents",
			kName:       "empty_tk",
			parents:     []toolkit.Parent{},
			expectCount: 0,
			expectNames: []string{},
		},
		{
			name:        "one parent",
			kName:       "one_parent_tk",
			parents:     []toolkit.Parent{parent1},
			expectCount: 1,
			expectNames: []string{"parent1"},
		},
		{
			name:        "two parents",
			kName:       "two_parent_tk",
			parents:     []toolkit.Parent{parent1, parent2},
			expectCount: 2,
			expectNames: []string{"parent1", "parent2"},
		},
		{
			name:        "nil parent ignored",
			kName:       "nil_ignored_tk",
			parents:     []toolkit.Parent{parent1, nil, parent2}, // Add nil parent
			expectCount: 2,
			expectNames: []string{"parent1", "parent2"},
		},
		{
			name:        "duplicate parent overwrites",
			kName:       "dup_overwrite_tk",
			parents:     []toolkit.Parent{parent1, parent2, parent1}, // Duplicate parent1
			expectCount: 2,
			expectNames: []string{"parent1", "parent2"},
			// We can't easily assert which parent1 was kept without reflection/more complex setup
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tk := toolkit.New(tc.kName, tc.parents...)
			require.NotNil(t, tk)
			assert.Equal(t, tc.kName, tk.GetToolkitName())

			// Check GetToolkitDescription contains expected parent names
			desc := tk.GetToolkitDescription()

			// Assert correct number of parent blocks (approximate check)
			expectedParentBlocks := tc.expectCount
			actualParentBlocks := strings.Count(desc, "<parent name=")
			assert.Equal(t, expectedParentBlocks, actualParentBlocks, "Description should contain the correct number of parent blocks")

			// Assert specific parent names are present
			for _, name := range tc.expectNames {
				expectStr := fmt.Sprintf(`<parent name="%s"`, name)
				assert.Contains(t, desc, expectStr, "Description should contain parent name: %s", name)
			}
		})
	}
}

// --- Test HandleToolKit ---

func TestHandleToolKit_Success(t *testing.T) {
	// Setup toolkit instance
	parent1 := createTestParent(t, "parent1",
		createTestChildFn(t, "c1a", "r1a", false),
		createTestChildFn(t, "c1b", "r1b", false),
	)
	parent2 := createTestParent(t, "parent2",
		createTestChildFn(t, "c2a", "r2a", false),
	)
	tk := toolkit.New("test_handle_success", parent1, parent2)
	require.NotNil(t, tk)

	// Input request JSON
	inputJSON := `{
		"name": "toolkit",
		"parents": [
			{
				"name": "parent1",
				"childs": [
					{"name": "c1b", "args": {"val": "v1b"}},
					{"name": "c1a", "args": {"val": "v1a"}}
				]
			},
			{
				"name": "parent2",
				"childs": [
					{"name": "c2a", "args": {"val": "v2a"}}
				]
			}
		]
	}`

	ctx := context.Background()
	resp, err := tk.HandleToolKit(ctx, json.RawMessage(inputJSON))
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "test_handle_success", resp.Name)
	require.Len(t, resp.Responses, 2)

	// Check Parent 1 response
	pr1 := resp.Responses[0]
	assert.Equal(t, "parent1", pr1.Name)
	require.Len(t, pr1.ChildsResponses, 2)
	cr1b := pr1.ChildsResponses[0]
	assert.Equal(t, "c1b", cr1b.Name)
	assert.Equal(t, testResp{Res: "r1b:v1b"}, cr1b.Response)
	cr1a := pr1.ChildsResponses[1]
	assert.Equal(t, "c1a", cr1a.Name)
	assert.Equal(t, testResp{Res: "r1a:v1a"}, cr1a.Response)

	// Check Parent 2 response
	pr2 := resp.Responses[1]
	assert.Equal(t, "parent2", pr2.Name)
	require.Len(t, pr2.ChildsResponses, 1)
	cr2a := pr2.ChildsResponses[0]
	assert.Equal(t, "c2a", cr2a.Name)
	assert.Equal(t, testResp{Res: "r2a:v2a"}, cr2a.Response)
}

func TestHandleToolKit_ParseError(t *testing.T) {
	tk := toolkit.New("test_parse_error") // No parents needed
	require.NotNil(t, tk)

	inputJSON := `{"invalid_json...`

	ctx := context.Background()
	resp, err := tk.HandleToolKit(ctx, json.RawMessage(inputJSON))
	require.Error(t, err) // Expecting the raw unmarshal error
	require.NotNil(t, resp)
	assert.Equal(t, "toolkit_request_parse_error", resp.Name)
	require.Len(t, resp.Responses, 1)
	pr := resp.Responses[0]
	assert.Equal(t, "_parse_error", pr.Name)
	require.Len(t, pr.ChildsResponses, 1)
	cr := pr.ChildsResponses[0]
	assert.Equal(t, "_input_error", cr.Name)
	tkErr, ok := cr.Response.(toolkit.ToolKitError)
	require.True(t, ok, "Expected response to be ToolKitError")
	assert.Equal(t, "invalid_input_json", tkErr.Code)
}

func TestHandleToolKit_ParentNotFound(t *testing.T) {
	parent1 := createTestParent(t, "parent1")
	tk := toolkit.New("test_p_notfound", parent1)
	require.NotNil(t, tk)

	inputJSON := `{
		"name": "toolkit",
		"parents": [
			{"name": "non_existent_parent", "childs": []}
		]
	}`

	ctx := context.Background()
	resp, err := tk.HandleToolKit(ctx, json.RawMessage(inputJSON))
	require.NoError(t, err) // HandleToolKit itself doesn't error here
	require.NotNil(t, resp)
	require.Len(t, resp.Responses, 1)

	pr := resp.Responses[0]
	assert.Equal(t, "non_existent_parent", pr.Name)
	require.Len(t, pr.ChildsResponses, 1)
	cr := pr.ChildsResponses[0]
	assert.Equal(t, "_parent_error", cr.Name)
	tkErr, ok := cr.Response.(toolkit.ToolKitError)
	require.True(t, ok, "Expected response to be ToolKitError")
	assert.Equal(t, "parent_not_found", tkErr.Code)
}

func TestHandleToolKit_ChildNotFound(t *testing.T) {
	parent1 := createTestParent(t, "parent1", createTestChildFn(t, "c1a", "r1a", false))
	tk := toolkit.New("test_c_notfound", parent1)
	require.NotNil(t, tk)

	inputJSON := `{
		"name": "toolkit",
		"parents": [
			{"name": "parent1", "childs": [{"name": "non_existent_child", "args": {}}]}
		]
	}`

	ctx := context.Background()
	resp, err := tk.HandleToolKit(ctx, json.RawMessage(inputJSON))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Responses, 1)

	pr := resp.Responses[0]
	assert.Equal(t, "parent1", pr.Name)
	require.Len(t, pr.ChildsResponses, 1)
	cr := pr.ChildsResponses[0]
	assert.Equal(t, "non_existent_child", cr.Name)
	tkErr, ok := cr.Response.(toolkit.ToolKitError)
	require.True(t, ok, "Expected response to be ToolKitError")
	assert.Equal(t, "child_not_found", tkErr.Code)
}

func TestHandleToolKit_ChildHandlerError(t *testing.T) {
	parent1 := createTestParent(t, "parent1", createTestChildFn(t, "c1a_err", "r1a", true)) // This child will error
	tk := toolkit.New("test_c_err", parent1)
	require.NotNil(t, tk)

	inputJSON := `{
		"name": "toolkit",
		"parents": [
			{"name": "parent1", "childs": [{"name": "c1a_err", "args": {"val":"v1"}}]}
		]
	}`

	ctx := context.Background()
	resp, err := tk.HandleToolKit(ctx, json.RawMessage(inputJSON))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Responses, 1)

	pr := resp.Responses[0]
	require.Len(t, pr.ChildsResponses, 1)
	cr := pr.ChildsResponses[0]
	assert.Equal(t, "c1a_err", cr.Name)
	tkErr, ok := cr.Response.(toolkit.ToolKitError)
	require.True(t, ok, "Expected response to be ToolKitError")
	assert.Equal(t, "handler_execution_error", tkErr.Code)
}

func TestHandleToolKit_ChildUnmarshalError(t *testing.T) {
	parent1 := createTestParent(t, "parent1", createTestChildFn(t, "c1a", "r1a", false))
	tk := toolkit.New("test_c_unmarshal_err", parent1)
	require.NotNil(t, tk)

	// Use JSON with incorrect type for the expected field
	inputJSON := `{
		"name": "toolkit",
		"parents": [
			{"name": "parent1", "childs": [{"name": "c1a", "args": {"val": 123}}]}
		]
	}`

	ctx := context.Background()
	resp, err := tk.HandleToolKit(ctx, json.RawMessage(inputJSON))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Responses, 1)

	pr := resp.Responses[0]
	require.Len(t, pr.ChildsResponses, 1)
	cr := pr.ChildsResponses[0]
	assert.Equal(t, "c1a", cr.Name)
	tkErr, ok := cr.Response.(toolkit.ToolKitError)
	require.True(t, ok, "Expected response to be ToolKitError")
	assert.Equal(t, "invalid_arguments", tkErr.Code)
}

// --- Test GetToolkitDescription ---

func TestGetToolkitDescription(t *testing.T) {
	// Setup toolkit instance
	parent1 := createTestParent(t, "p1",
		createTestChildFn(t, "c1a", "r1a", false),
	)
	parent2 := createTestParent(t, "p2",
		createTestChildFn(t, "c2a", "r2a", false),
		createTestChildFn(t, "c2b", "r2b", false),
	)
	emptyParent := createTestParent(t, "emptyP")

	tests := []struct {
		name            string
		kName           string
		parents         []toolkit.Parent
		expectToContain []string
	}{
		{
			name:    "no parents",
			kName:   "tk_empty",
			parents: []toolkit.Parent{},
			expectToContain: []string{
				`<toolkit name="tk_empty">`,
				`</toolkit>`,
				`Below is the list of available <parents> and their <childs>:`,
			},
		},
		{
			name:    "with parents and children",
			kName:   "tk_full",
			parents: []toolkit.Parent{parent1, parent2, emptyParent},
			expectToContain: []string{
				`<toolkit name="tk_full">`,
				`<parent name="p1" description="desc_p1">`,  // Parent 1 start
				`<child name="c1a" description="desc_c1a">`, // Child 1a
				`"properties":{"val":`,                      // Schema snippet for c1a
				`</parent>`,                                 // Parent 1 end
				`<parent name="p2" description="desc_p2">`,  // Parent 2 start
				`<child name="c2a" description="desc_c2a">`, // Child 2a
				`<child name="c2b" description="desc_c2b">`, // Child 2b
				`</parent>`, // Parent 2 end
				`<parent name="emptyP" description="desc_emptyP">`, // Empty Parent start
				`</parent>`,  // Empty Parent end
				`</toolkit>`, // Toolkit end
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tk := toolkit.New(tc.kName, tc.parents...)
			require.NotNil(t, tk)

			desc := tk.GetToolkitDescription()
			// fmt.Println(desc) // Uncomment for debugging

			for _, expected := range tc.expectToContain {
				assert.Contains(t, desc, expected, "Description should contain expected substring")
			}
		})
	}
}

// --- Test GetToolkitSchema ---

func TestGetToolkitSchema(t *testing.T) {
	tk := toolkit.New("test_schema")
	require.NotNil(t, tk)

	// Test known provider
	anthropicSchema := tk.GetToolkitSchema("anthropic")
	assert.NotNil(t, anthropicSchema, "Schema for known provider 'anthropic' should not be nil")

	// Check the actual type returned by the jsonschema library
	schemaPtr, ok := anthropicSchema.(*jsonschema.Schema)
	require.True(t, ok, "Anthropic schema should be a *jsonschema.Schema")
	assert.Equal(t, "object", schemaPtr.Type, "Expected schema type to be object")
	assert.NotNil(t, schemaPtr.Properties, "Expected schema properties to be non-nil")

	// Test unknown provider (should default/log warning - can't test log easily)
	unknownSchema := tk.GetToolkitSchema("unknown_provider")
	assert.NotNil(t, unknownSchema, "Schema for unknown provider should default and not be nil")
	assert.Equal(t, anthropicSchema, unknownSchema, "Schema for unknown provider should default to Anthropic schema")
}

// Removed TODO placeholders
