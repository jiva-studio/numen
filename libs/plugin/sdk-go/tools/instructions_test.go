package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

func serveRegistry(t *testing.T, registry *Registry) Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginToolServiceHandler(registryHandler{registry: registry}))
	web := httptest.NewServer(mux)
	t.Cleanup(web.Close)
	return NewClient(hostcall.Endpoint{BaseURL: web.URL})
}

const (
	findSentence  = "Find the fixture with fixture_find before you write it."
	writeSentence = "Write the fixture with fixture_write when it changes."
)

func newRegistryWithInstructions(t *testing.T) *Registry {
	t.Helper()
	registry := New()
	for _, spec := range []Spec{
		{
			Name: "fixture_find", Description: "Finds the fixture.", Effect: ReadOnly,
			Instructions: []string{findSentence, writeSentence},
		},
		{Name: "fixture_write", Description: "Writes the fixture.", Effect: ReadWrite, Instructions: []string{writeSentence}},
		{Name: "fixture_list", Description: "Lists the fixtures.", Effect: ReadOnly},
	} {
		if err := Add(registry, spec, func(context.Context, greetIn) (greetOut, error) { return greetOut{}, nil }); err != nil {
			t.Fatal(err)
		}
	}
	return registry
}

func getToolInstructions(listing Listing) map[string][]string {
	byTool := map[string][]string{}
	for _, one := range listing.Tools {
		byTool[one.GetName()] = one.Instructions
	}
	return byTool
}

func TestAToolListsTheInstructionsOfItsSpecOnTheWire(t *testing.T) {
	res, err := registryHandler{registry: newRegistryWithInstructions(t)}.ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	specs := res.Msg.GetTools()
	if len(specs) != 3 {
		t.Fatalf("%d tools listed", len(specs))
	}
	if got := specs[0].GetInstructions(); !slices.Equal(got, []string{findSentence, writeSentence}) {
		t.Errorf("fixture_find lists the instructions %q", got)
	}
	if got := specs[1].GetInstructions(); !slices.Equal(got, []string{writeSentence}) {
		t.Errorf("fixture_write lists the instructions %q", got)
	}
	if got := specs[2].GetInstructions(); len(got) != 0 {
		t.Errorf("fixture_list has no instructions in its spec and lists %q", got)
	}
}

func TestToolInstructionsAreSentToAClientOverTheWire(t *testing.T) {
	listing, err := serveRegistry(t, newRegistryWithInstructions(t)).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	byTool := getToolInstructions(listing)
	if got := byTool["fixture_find"]; !slices.Equal(got, []string{findSentence, writeSentence}) {
		t.Errorf("the client read the instructions of fixture_find as %q", got)
	}
	if got := byTool["fixture_write"]; !slices.Equal(got, []string{writeSentence}) {
		t.Errorf("the client read the instructions of fixture_write as %q", got)
	}
	if got := byTool["fixture_list"]; len(got) != 0 {
		t.Errorf("the client read the instructions of fixture_list as %q", got)
	}
}
