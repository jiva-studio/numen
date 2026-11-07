package service

import (
	"encoding/json"
	"reflect"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// schemaCase is a schema the host lists for a tool, used as both its input and its output schema,
// and what an MCP client is given for each. The same table is in
// libs/core/internal/adapter/plugin/agenttoolsbridge/declare_tool_test.go: both readers declare a
// tool identically.
type schemaCase struct {
	name   string
	schema []byte
	// input is the input schema the client is given.
	input string
	// isInputAsListed is true where the input schema is the listed bytes unchanged.
	isInputAsListed bool
	// output is the output schema the client is given, and empty where it is left out.
	output string
}

var schemaCases = []schemaCase{
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

// assertDeclaredSchemas fails the test unless the declared tool carries the schemas of the case.
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

func TestTheMCPServerDeclaresTheSchemasOfATool(t *testing.T) {
	t.Parallel()
	for _, tc := range schemaCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			declared := createTool(&tools.ToolInfo{
				Name: "pond_read", Effect: tools.ReadOnly, InputSchema: tc.schema, OutputSchema: tc.schema,
			})
			assertDeclaredSchemas(t, tc, declared)
		})
	}
}

// hintCase is a tool whose hints say otherwise than its effect, step and required permissions, and
// the annotations an MCP client is given. The same table is in
// libs/core/internal/adapter/plugin/agenttoolsbridge/declare_tool_test.go.
type hintCase struct {
	name     string
	effect   tools.Effect
	step     tools.Step
	requires []string
	hints    tools.Hints
	// readOnly, idempotent, destructive and openWorld are the annotations the client is given.
	readOnly, idempotent, destructive, openWorld bool
}

var hintCases = []hintCase{
	{
		name: "a removing tool that needs the network with the hints of a local read", effect: tools.ReadWrite,
		step: tools.Remove, requires: []string{"network:outbound"},
		hints:    tools.Hints{ReadOnly: true, Idempotent: true},
		readOnly: true, idempotent: true,
	},
	{
		name: "a local read with the hints of a removing tool that needs the network", effect: tools.ReadOnly,
		hints:       tools.Hints{Destructive: true, OpenWorld: true},
		destructive: true, openWorld: true,
	},
	{
		name: "a moving tool with the destructive hint", effect: tools.ReadWrite, step: tools.Move,
		hints:       tools.Hints{Destructive: true},
		destructive: true,
	},
	{
		name: "a read that needs the network with no hints", effect: tools.ReadOnly,
		requires: []string{"network:outbound"},
	},
	{
		name: "a removing tool with every hint", effect: tools.ReadWrite, step: tools.Remove,
		hints:    tools.Hints{ReadOnly: true, Destructive: true, Idempotent: true, OpenWorld: true},
		readOnly: true, idempotent: true, destructive: true, openWorld: true,
	},
}

func (tc hintCase) describeWant() string {
	return describeAnnotations(&sdk.ToolAnnotations{
		ReadOnlyHint: tc.readOnly, IdempotentHint: tc.idempotent, DestructiveHint: &tc.destructive, OpenWorldHint: &tc.openWorld,
	})
}

func TestTheMCPServerAnnotatesAToolFromItsHintsOnly(t *testing.T) {
	t.Parallel()
	listing := make([]*tools.ToolInfo, 0, len(hintCases))
	byName := map[string]hintCase{}
	for i, tc := range hintCases {
		name := "pond_tool_" + string(rune('a'+i))
		byName[name] = tc
		listing = append(listing, &tools.ToolInfo{
			Name: name, Description: tc.name, InputSchema: []byte(`{"type":"object"}`),
			Effect: tc.effect, Step: tc.step, Requires: tc.requires, Hints: tc.hints,
		})
	}
	srv, err := NewServer(Config{}, &fakeToolClient{tools: listing})
	if err != nil {
		t.Fatal(err)
	}

	listed := listTools(t, connectInMemory(t, srv, nil))

	if len(listed) != len(hintCases) {
		t.Fatalf("tools/list returned %d tools, want %d", len(listed), len(hintCases))
	}
	for _, tool := range listed {
		tc, isKnown := byName[tool.Name]
		if !isKnown {
			t.Errorf("tools/list returned the tool %s", tool.Name)
			continue
		}
		if got, want := describeAnnotations(tool.Annotations), tc.describeWant(); got != want {
			t.Errorf("%s: the tool is annotated %s, want %s", tc.name, got, want)
		}
	}
}
