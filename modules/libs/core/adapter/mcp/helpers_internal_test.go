package mcp

import (
	"reflect"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFillResultKeepsExistingContent(t *testing.T) {
	retained := &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: "x"}}}
	if got := fillResult(retained, struct{}{}); got != retained || got.StructuredContent != nil {
		t.Fatalf("a result with content must be returned as it is, got %+v", got)
	}
}

func TestFillResultSetsContentByOutputShape(t *testing.T) {
	object := fillResult(nil, struct{ A int }{A: 1})
	if object.StructuredContent == nil || len(object.Content) != 0 {
		t.Fatalf("an object is structured content, got %+v", object)
	}
	text := fillResult(nil, []int{1})
	if text.StructuredContent != nil || len(text.Content) != 1 {
		t.Fatalf("a non-object is text content, got %+v", text)
	}
}

func TestBuildOutputSchemaOnlyForStructs(t *testing.T) {
	if buildOutputSchema(reflect.TypeFor[any]()) != nil {
		t.Fatal("any has no output schema")
	}
	if buildOutputSchema(reflect.TypeFor[[]string]()) != nil {
		t.Fatal("a slice has no output schema")
	}
	if buildOutputSchema(reflect.TypeFor[*struct{ A int }]()) == nil {
		t.Fatal("a pointer to a struct has an output schema")
	}
}
