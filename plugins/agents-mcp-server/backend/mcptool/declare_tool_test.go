package mcptool

import (
	"encoding/json"
	"reflect"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

// schemaCase is a schema the host lists for a tool, used as both its input and its output schema,
// and what an MCP client is given for each. The same table is in
// libs/core/kernel/internal/adapter/plugin/agenttoolsbridge/declare_tool_test.go: both readers declare a
// tool identically.
type schemaCase struct {
	name            string
	schema          []byte
	input           string
	isInputAsListed bool
	output          string
}

func getSchemaCases() []schemaCase {
	return []schemaCase{
		{
			name:            "an object of type object",
			schema:          []byte(`{"type":"object","properties":{"title":{"type":"string"},"author":{"type":"string"}},"required":["title"]}`),
			input:           `{"type":"object","properties":{"title":{"type":"string"},"author":{"type":"string"}},"required":["title"]}`,
			isInputAsListed: true,
			output:          `{"type":"object","properties":{"title":{"type":"string"},"author":{"type":"string"}},"required":["title"]}`,
		},
		{
			name:   "an object without a type",
			schema: []byte(`{"properties":{"pond":{"type":"string"}},"required":["pond"]}`),
			input:  `{"type":"object","properties":{"pond":{"type":"string"}},"required":["pond"]}`,
		},
		{
			name:   "an object of type string",
			schema: []byte(`{"type":"string","properties":{"pond":{"type":"string"}}}`),
			input:  `{"type":"object","properties":{"pond":{"type":"string"}}}`,
		},
		{
			name:   "an object whose type is a list",
			schema: []byte(`{"type":["null","object"],"properties":{"pond":{"type":"string"}}}`),
			input:  `{"type":"object","properties":{"pond":{"type":"string"}}}`,
		},
		{
			name:   "an object of type Object in capitals",
			schema: []byte(`{"type":"Object"}`),
			input:  `{"type":"object"}`,
		},
		{name: "the empty object", schema: []byte(`{}`), input: `{"type":"object"}`},
		{name: "no bytes", schema: nil, input: `{"type":"object"}`},
		{name: "invalid JSON", schema: []byte(`{"type":`), input: `{"type":"object"}`},
		{name: "null", schema: []byte(`null`), input: `{"type":"object"}`},
		{name: "an array", schema: []byte(`[{"type":"object"}]`), input: `{"type":"object"}`},
		{name: "a string", schema: []byte(`"object"`), input: `{"type":"object"}`},
		{name: "a number", schema: []byte(`7`), input: `{"type":"object"}`},
	}
}

func assertDeclaredSchemas(t *testing.T, tc schemaCase, declared *sdk.Tool) {
	t.Helper()
	input, err := json.Marshal(declared.InputSchema)
	if err != nil {
		t.Fatalf("the input schema %v cannot be encoded: %v", declared.InputSchema, err)
	}
	if tc.isInputAsListed && string(input) != tc.input {
		t.Errorf("the input schema is %s, want the listed bytes %s", input, tc.input)
	}
	if !isSameJSON(input, []byte(tc.input)) {
		t.Errorf("the input schema is %s, want %s", input, tc.input)
	}
	if tc.output == "" {
		if declared.OutputSchema != nil {
			t.Errorf("the output schema is %s, want it left out", declared.OutputSchema)
		}
		return
	}
	output, err := json.Marshal(declared.OutputSchema)
	if err != nil || !isSameJSON(output, []byte(tc.output)) {
		t.Errorf("the output schema is %s (%v), want %s", output, err, tc.output)
	}
}

func isSameJSON(a, b []byte) bool {
	var left, right any
	return json.Unmarshal(a, &left) == nil && json.Unmarshal(b, &right) == nil && reflect.DeepEqual(left, right)
}

func TestNewToolDeclaresTheSchemasOfATool(t *testing.T) {
	t.Parallel()
	for _, tc := range getSchemaCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			declared := NewTool(&tools.ToolInfo{
				Name: "pond_read", Effect: tools.ReadOnly, InputSchema: tc.schema, OutputSchema: tc.schema,
			})
			assertDeclaredSchemas(t, tc, declared)
		})
	}
}
