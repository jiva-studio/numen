package mcp

import (
	"reflect"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
)

func TestGetCardRangeClampsToTheDeck(t *testing.T) {
	deck := make([]format.Card, 3)
	cases := []struct {
		name        string
		from, limit int
		want        int
	}{
		{"whole deck by default", 0, 0, 3},
		{"limit cuts the run", 0, 2, 2},
		{"negative start begins at the first card", -4, 0, 3},
		{"start past the end is empty", 9, 0, 0},
		{"limit past the end stops at the end", 1, 50, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := len(getCardRange(deck, c.from, c.limit)); got != c.want {
				t.Fatalf("got %d cards, want %d", got, c.want)
			}
		})
	}
}

func TestFillResultKeepsExistingContent(t *testing.T) {
	kept := &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: "x"}}}
	if got := fillResult(kept, struct{}{}); got != kept || got.StructuredContent != nil {
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
