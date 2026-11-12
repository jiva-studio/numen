package service

import (
	"fmt"
	"strconv"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func describeAnnotations(annotations *sdk.ToolAnnotations) string {
	if annotations == nil {
		return "no annotations"
	}
	describe := func(hint *bool) string {
		if hint == nil {
			return "nil"
		}
		return strconv.FormatBool(*hint)
	}
	return fmt.Sprintf("readOnly=%t idempotent=%t destructive=%s openWorld=%s",
		annotations.ReadOnlyHint, annotations.IdempotentHint,
		describe(annotations.DestructiveHint), describe(annotations.OpenWorldHint))
}

// hintCase is a tool whose hints say otherwise than its effect, step and required permissions, and
// the annotations an MCP client is given. The same table is in
// libs/core/kernel/internal/adapter/plugin/agenttoolsbridge/declare_tool_test.go.
type hintCase struct {
	name                                         string
	effect                                       tools.Effect
	step                                         tools.Step
	requires                                     []string
	hints                                        tools.Hints
	readOnly, idempotent, destructive, openWorld bool
}

func getHintCases() []hintCase {
	return []hintCase{
		{
			name: "a removing tool that needs the network with the hints of a local read", effect: tools.ReadWrite,
			step: tools.Remove, requires: []string{"link:open"},
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
			requires: []string{"link:open"},
		},
		{
			name: "a removing tool with every hint", effect: tools.ReadWrite, step: tools.Remove,
			hints:    tools.Hints{ReadOnly: true, Destructive: true, Idempotent: true, OpenWorld: true},
			readOnly: true, idempotent: true, destructive: true, openWorld: true,
		},
	}
}

func (tc hintCase) describeWant() string {
	return describeAnnotations(&sdk.ToolAnnotations{
		ReadOnlyHint: tc.readOnly, IdempotentHint: tc.idempotent, DestructiveHint: &tc.destructive, OpenWorldHint: &tc.openWorld,
	})
}

func TestTheMCPServerAnnotatesAToolFromItsHintsOnly(t *testing.T) {
	t.Parallel()
	hintCases := getHintCases()
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

	declared := listTools(t, connectInMemory(t, srv, nil))

	if len(declared) != len(hintCases) {
		t.Fatalf("tools/list returned %d tools, want %d", len(declared), len(hintCases))
	}
	for _, tool := range declared {
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
