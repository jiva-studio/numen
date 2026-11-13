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

// annotationCase is a tool's effect and required permissions and the annotations an MCP client is
// given.
type annotationCase struct {
	name                                         string
	effect                                       tools.Effect
	requires                                     []string
	readOnly, idempotent, destructive, openWorld bool
}

func getAnnotationCases() []annotationCase {
	return []annotationCase{
		{name: "a local read", effect: tools.ReadOnly, readOnly: true, idempotent: true},
		{name: "a local write", effect: tools.ReadWrite, destructive: true},
		{
			name: "a read that needs the network", effect: tools.ReadOnly, requires: []string{"link:open"},
			readOnly: true, idempotent: true, openWorld: true,
		},
		{
			name: "a write that needs the network and to manage vaults", effect: tools.ReadWrite,
			requires: []string{"vault:manage", "link:open"}, destructive: true, openWorld: true,
		},
		{name: "a write that needs to manage vaults", effect: tools.ReadWrite, requires: []string{"vault:manage"}, destructive: true},
		{name: "a tool of no effect", effect: 0},
	}
}

func (tc annotationCase) describeWant() string {
	return describeAnnotations(&sdk.ToolAnnotations{
		ReadOnlyHint: tc.readOnly, IdempotentHint: tc.idempotent, DestructiveHint: &tc.destructive, OpenWorldHint: &tc.openWorld,
	})
}

func TestTheMCPServerAnnotatesAToolFromItsEffectAndRequiredPermissions(t *testing.T) {
	t.Parallel()
	annotationCases := getAnnotationCases()
	listing := make([]*tools.ToolInfo, 0, len(annotationCases))
	byName := map[string]annotationCase{}
	for i, tc := range annotationCases {
		name := "pond_tool_" + string(rune('a'+i))
		byName[name] = tc
		listing = append(listing, &tools.ToolInfo{
			Name: name, Description: tc.name, InputSchema: []byte(`{"type":"object"}`),
			Effect: tc.effect, Requires: tc.requires,
		})
	}
	srv, err := NewServer(Config{}, &fakeToolClient{tools: listing})
	if err != nil {
		t.Fatal(err)
	}

	declared := listTools(t, connectInMemory(t, srv, nil))

	if len(declared) != len(annotationCases) {
		t.Fatalf("tools/list returned %d tools, want %d", len(declared), len(annotationCases))
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
