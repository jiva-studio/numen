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
