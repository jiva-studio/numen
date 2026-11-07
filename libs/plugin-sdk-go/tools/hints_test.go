package tools

import (
	"context"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// newHintedRegistry adds one tool for each combination of effect, step and required permission that
// decides a hint.
func newHintedRegistry(t *testing.T) *Registry {
	t.Helper()
	registry := New()
	for _, spec := range []Spec{
		{Name: "pond_read", Effect: ReadOnly},
		{Name: "pond_search", Effect: ReadOnly, Step: Search},
		{Name: "pond_fetch", Effect: ReadOnly, Requires: []string{"network:outbound"}},
		{Name: "pond_write", Effect: ReadWrite},
		{Name: "pond_drain", Effect: ReadWrite, Step: Remove, Requires: []string{"vault:manage"}},
		{Name: "pond_rename", Effect: ReadWrite, Step: Move},
		{Name: "pond_import", Effect: ReadWrite, Requires: []string{"network:outbound"}},
	} {
		if err := Add(registry, spec, func(context.Context, greetIn) (greetOut, error) { return greetOut{}, nil }); err != nil {
			t.Fatal(err)
		}
	}
	return registry
}

var wantHints = map[string]Hints{
	"pond_read":   {ReadOnly: true, Idempotent: true},
	"pond_search": {ReadOnly: true, Idempotent: true},
	"pond_fetch":  {ReadOnly: true, Idempotent: true, OpenWorld: true},
	"pond_write":  {},
	"pond_drain":  {Destructive: true},
	"pond_rename": {},
	"pond_import": {OpenWorld: true},
}

func TestToolAddSetsTheHintsOfTheToolOnTheWire(t *testing.T) {
	res, err := registryHandler{registry: newHintedRegistry(t)}.ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Msg.GetTools()) != len(wantHints) {
		t.Fatalf("%d tools listed, want %d", len(res.Msg.GetTools()), len(wantHints))
	}
	for _, tool := range res.Msg.GetTools() {
		want := wantHints[tool.GetName()]
		hints := tool.GetHints()
		got := Hints{
			ReadOnly: hints.GetReadOnly(), Destructive: hints.GetDestructive(),
			Idempotent: hints.GetIdempotent(), OpenWorld: hints.GetOpenWorld(),
		}
		if hints == nil || got != want {
			t.Errorf("%s lists the hints %v, want %+v", tool.GetName(), hints, want)
		}
	}
}

func TestToolHintsAreReadFromTheRegistryAndOverTheWire(t *testing.T) {
	registry := newHintedRegistry(t)
	local, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	remote, err := serveRegistry(t, registry).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	for source, listing := range map[string]Listing{"the registry": local, "the client": remote} {
		if len(listing.Tools) != len(wantHints) {
			t.Fatalf("%s lists %d tools, want %d", source, len(listing.Tools), len(wantHints))
		}
		for _, info := range listing.Tools {
			if got, want := info.GetHints(), wantHints[info.GetName()]; got != want {
				t.Errorf("%s reads the hints of %s as %+v, want %+v", source, info.GetName(), got, want)
			}
		}
	}
}

func TestServiceHandlerToolHintsAreSentOverTheWire(t *testing.T) {
	sent := []*ToolInfo{
		{Name: "pond_drain", Effect: ReadOnly, Hints: Hints{Destructive: true, OpenWorld: true}},
		{Name: "pond_read", Effect: ReadWrite, Step: Remove, Hints: Hints{ReadOnly: true, Idempotent: true}},
		{Name: "pond_write", Effect: ReadWrite},
	}

	listing, err := serveHandler(t, listingHandler{listing: Listing{Tools: sent}}).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Tools) != len(sent) {
		t.Fatalf("the client read %d tools, want %d", len(listing.Tools), len(sent))
	}
	for i, info := range listing.Tools {
		if got, want := info.GetHints(), sent[i].Hints; got != want {
			t.Errorf("the client read the hints of %s as %+v, want %+v", info.GetName(), got, want)
		}
	}
}

func TestTheHintsOfANilToolInfoAreUnset(t *testing.T) {
	var info *ToolInfo
	if got := info.GetHints(); got != (Hints{}) {
		t.Errorf("a nil tool info has the hints %+v", got)
	}
}

func TestToolAddWritesAnInputSchemaOfTypeObject(t *testing.T) {
	registry := New()
	if err := Add(registry, Spec{Name: "pond_count", Effect: ReadOnly}, func(context.Context, int) (int, error) { return 0, nil }); err != nil {
		t.Fatal(err)
	}
	if err := Add(registry, Spec{Name: "pond_read", Effect: ReadOnly}, func(context.Context, *greetIn) (greetOut, error) {
		return greetOut{}, nil
	}); err != nil {
		t.Fatal(err)
	}
	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range listing.Tools {
		var schema map[string]any
		if err := json.Unmarshal(info.GetInputSchema(), &schema); err != nil || schema["type"] != "object" {
			t.Errorf("%s lists the input schema %s (%v), want an object of type object", info.GetName(), info.GetInputSchema(), err)
		}
	}
}
