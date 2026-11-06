package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"
)

const lanternInstructions = "Light the lantern with lantern_light before you read the map, and name every street you pass."

type listingHandler struct {
	listing Listing
}

func (h listingHandler) ListTools(context.Context, http.Header) (Listing, error) {
	return h.listing, nil
}

func (h listingHandler) InvokeTool(context.Context, http.Header, Call) (*CallResponse, error) {
	return &CallResponse{}, nil
}

func serveRegistry(t *testing.T, registry *Registry) Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginToolServiceHandler(registryHandler{registry: registry}))
	web := httptest.NewServer(mux)
	t.Cleanup(web.Close)
	return NewClient(hostcall.Endpoint{BaseURL: web.URL})
}

func serveHandler(t *testing.T, handler ServiceHandler) Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(NewServiceHandler(handler))
	web := httptest.NewServer(mux)
	t.Cleanup(web.Close)
	return NewClient(hostcall.Endpoint{BaseURL: web.URL})
}

func TestARegistryListsItsInstructionsNextToItsTools(t *testing.T) {
	registry := newRegistry(t)
	registry.SetInstructions(lanternInstructions)

	listing, err := registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if listing.Instructions != lanternInstructions {
		t.Errorf("the instructions are %q, want %q", listing.Instructions, lanternInstructions)
	}
	if len(listing.Tools) != 1 || listing.Tools[0].GetName() != "greet" {
		t.Errorf("the tools are %v", listing.Tools)
	}
}

func TestARegistryWithoutInstructionsListsNone(t *testing.T) {
	listing, err := newRegistry(t).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if listing.Instructions != "" {
		t.Errorf("a registry given no instructions lists %q", listing.Instructions)
	}
	if len(listing.Tools) != 1 {
		t.Errorf("%d tools listed", len(listing.Tools))
	}
}

func TestTheInstructionsOfARegistryReachAClientOverTheWire(t *testing.T) {
	registry := newRegistry(t)
	registry.SetInstructions(lanternInstructions)

	listing, err := serveRegistry(t, registry).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if listing.Instructions != lanternInstructions {
		t.Errorf("the client read the instructions %q, want %q", listing.Instructions, lanternInstructions)
	}
	if len(listing.Tools) != 1 || listing.Tools[0].GetName() != "greet" {
		t.Errorf("the client read the tools %v", listing.Tools)
	}
}

func TestARegistryWithoutInstructionsReachesAClientWithNone(t *testing.T) {
	listing, err := serveRegistry(t, newRegistry(t)).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if listing.Instructions != "" {
		t.Errorf("the client read the instructions %q from a registry given none", listing.Instructions)
	}
}

func TestTheInstructionsOfAServiceHandlerReachAClientOverTheWire(t *testing.T) {
	handler := listingHandler{listing: Listing{
		Tools:        []*ToolInfo{{Name: "lantern_light", Description: "Lights the lantern."}},
		Instructions: lanternInstructions,
	}}

	listing, err := serveHandler(t, handler).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if listing.Instructions != lanternInstructions {
		t.Errorf("the client read the instructions %q, want %q", listing.Instructions, lanternInstructions)
	}
	if len(listing.Tools) != 1 || listing.Tools[0].GetName() != "lantern_light" {
		t.Errorf("the client read the tools %v", listing.Tools)
	}
}

const (
	lanternLightSentence = "Light the lantern with lantern_light before you read the map."
	lanternTrimSentence  = "Trim the wick with lantern_trim when the flame smokes."
)

func newInstructedRegistry(t *testing.T) *Registry {
	t.Helper()
	registry := New()
	for _, spec := range []Spec{
		{
			Name: "lantern_light", Description: "Lights the lantern.", Effect: ReadOnly,
			Instructions: []string{lanternLightSentence, lanternTrimSentence},
		},
		{Name: "lantern_trim", Description: "Trims the wick.", Effect: ReadWrite, Instructions: []string{lanternTrimSentence}},
		{Name: "lantern_look", Description: "Looks at the flame.", Effect: ReadOnly},
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
	res, err := registryHandler{registry: newInstructedRegistry(t)}.ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	specs := res.Msg.GetTools()
	if len(specs) != 3 {
		t.Fatalf("%d tools listed", len(specs))
	}
	if got := specs[0].GetInstructions(); !slices.Equal(got, []string{lanternLightSentence, lanternTrimSentence}) {
		t.Errorf("lantern_light lists the instructions %q", got)
	}
	if got := specs[1].GetInstructions(); !slices.Equal(got, []string{lanternTrimSentence}) {
		t.Errorf("lantern_trim lists the instructions %q", got)
	}
	if got := specs[2].GetInstructions(); len(got) != 0 {
		t.Errorf("lantern_look has no instructions in its spec and lists %q", got)
	}
	if res.Msg.GetInstructions() != "" {
		t.Errorf("a registry given no paragraph lists %q", res.Msg.GetInstructions())
	}
}

func TestTheInstructionsOfEachToolReachAClientOverTheWire(t *testing.T) {
	listing, err := serveRegistry(t, newInstructedRegistry(t)).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	byTool := getToolInstructions(listing)
	if got := byTool["lantern_light"]; !slices.Equal(got, []string{lanternLightSentence, lanternTrimSentence}) {
		t.Errorf("the client read the instructions of lantern_light as %q", got)
	}
	if got := byTool["lantern_trim"]; !slices.Equal(got, []string{lanternTrimSentence}) {
		t.Errorf("the client read the instructions of lantern_trim as %q", got)
	}
	if got := byTool["lantern_look"]; len(got) != 0 {
		t.Errorf("the client read the instructions of lantern_look as %q", got)
	}
}

func TestTheToolInstructionsOfAServiceHandlerReachAClientOverTheWire(t *testing.T) {
	handler := listingHandler{listing: Listing{Tools: []*ToolInfo{
		{Name: "lantern_light", Description: "Lights the lantern.", Effect: ReadOnly, Instructions: []string{lanternLightSentence}},
	}}}

	listing, err := serveHandler(t, handler).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got := getToolInstructions(listing)["lantern_light"]; !slices.Equal(got, []string{lanternLightSentence}) {
		t.Errorf("the client read the instructions of lantern_light as %q", got)
	}
}
