package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
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
