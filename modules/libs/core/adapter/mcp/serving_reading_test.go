package mcp_test

import (
	"context"
	"net/http"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
)

// The reviewer's window serves this endpoint, and an agent reaching it over the
// wire is offered the tools that read and no others. What a build registers is
// held elsewhere; this is what a caller finds on the port.
func TestAPortServingTheReadingToolsOffersNoOther(t *testing.T) {
	_, core := served(t)
	endpoint, err := mcp.ServeReadingHTTP(t.Context(), "127.0.0.1:0", "the-token", core, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { endpoint.Close(context.Background()) })

	client := sdk.NewClient(&sdk.Implementation{Name: "a test"}, nil)
	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{
		Endpoint:   endpoint.URL,
		HTTPClient: &http.Client{Transport: presenting{token: "the-token"}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	reading := map[string]bool{
		"note_search": true, "note_get": true, "note_read": true,
		"note_neighbourhood": true, "link_list": true, "source_list": true,
		"source_read": true, "card_stencils": true, "card_read": true,
		"vault_get": true,
	}
	found := map[string]bool{}
	for _, tool := range tools.Tools {
		if !reading[tool.Name] {
			t.Errorf("%s is served on the reading port", tool.Name)
		}
		found[tool.Name] = true
	}
	for name := range reading {
		if !found[name] {
			t.Errorf("%s is not served on the reading port", name)
		}
	}
}
