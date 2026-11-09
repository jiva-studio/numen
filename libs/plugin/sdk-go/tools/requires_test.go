package tools

import (
	"context"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

func newRegistryWithRequires(t *testing.T) *Registry {
	t.Helper()
	registry := New()
	for _, spec := range []Spec{
		{Name: "fixture_write", Description: "Writes the fixture.", Effect: ReadWrite, Requires: []string{"vault:manage"}},
		{Name: "fixture_read", Description: "Reads the fixture.", Effect: ReadOnly},
	} {
		if err := Add(registry, spec, func(context.Context, greetIn) (greetOut, error) { return greetOut{}, nil }); err != nil {
			t.Fatal(err)
		}
	}
	return registry
}

func TestAToolListsThePermissionsItsSpecRequires(t *testing.T) {
	res, err := registryHandler{registry: newRegistryWithRequires(t)}.ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	specs := res.Msg.GetTools()
	if len(specs) != 2 {
		t.Fatalf("%d tools listed", len(specs))
	}
	if got := specs[0].GetRequires(); !slices.Equal(got, []string{"vault:manage"}) {
		t.Errorf("fixture_write lists the requirements %q, want vault:manage", got)
	}
	if got := specs[1].GetRequires(); len(got) != 0 {
		t.Errorf("fixture_read requires nothing and lists %q", got)
	}
}

func TestToolRequiredPermissionsAreSentOverTheWire(t *testing.T) {
	listing, err := serveRegistry(t, newRegistryWithRequires(t)).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Tools) != 2 || !slices.Equal(listing.Tools[0].Requires, []string{"vault:manage"}) || len(listing.Tools[1].Requires) != 0 {
		t.Errorf("the client read the tools %+v", listing.Tools)
	}
}
