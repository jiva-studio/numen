package mcp_test

import (
	"net/http"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
)

// The reviewer's window serves this endpoint, and an agent reaching it over the
// wire is offered the tools that read and no others. What a build registers is
// held elsewhere; this is what a caller finds on the port.
func TestAPortServingTheReadingToolsOffersNoOther(t *testing.T) {
	_, core := newCore(t)
	endpoint, err := mcp.ServeReadingHTTP(t.Context(), "127.0.0.1:0", "the-token", core, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { endpoint.Close(t.Context()) })

	tools, err := listTools(t, endpoint.URL)
	if err != nil {
		t.Fatal(err)
	}

	onlyOffers(t, tools.Tools, "the reading port", []string{
		"note_search", "note_titles", "note_read", "note_neighbourhood", "link_list",
		"source_list", "source_read", "card_stencil_list", "card_read", "vault_get",
	})
}

// The window a person runs their cards in serves this endpoint. A card is
// written from there because that is what a person is doing; a deck and a
// stencil are what a vault is arranged into, and nothing there makes one.
func TestAPortServingTheReviewingToolsOffersNoOther(t *testing.T) {
	_, core := newCore(t)
	endpoint, err := mcp.ServeReviewingHTTP(t.Context(), "127.0.0.1:0", "the-token", core, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { endpoint.Close(t.Context()) })

	tools, err := listTools(t, endpoint.URL)
	if err != nil {
		t.Fatal(err)
	}

	onlyOffers(t, tools.Tools, "the reviewing port", []string{
		"note_search", "note_titles", "note_read", "note_neighbourhood", "link_list",
		"source_list", "source_read", "card_stencil_list", "card_read", "vault_get",
		"card_add", "card_edit", "card_value_remove", "card_remove",
		"card_section_add", "card_section_rename", "card_section_remove",
	})
}

// listTools is what an agent presenting the token is offered over the wire.
func listTools(t *testing.T, url string) (*sdk.ListToolsResult, error) {
	t.Helper()

	client := sdk.NewClient(&sdk.Implementation{Name: "a test"}, nil)
	session, err := client.Connect(t.Context(), &sdk.StreamableClientTransport{
		Endpoint:   url,
		HTTPClient: &http.Client{Transport: presenting{token: "the-token"}},
	}, nil)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { session.Close() })
	return session.ListTools(t.Context(), nil)
}

// onlyOffers is the tools an endpoint offers, as an exact set. The claim each of
// these ports makes is about what is absent, so a list that says what is
// present would pass with anything extra on it.
func onlyOffers(t *testing.T, offered []*sdk.Tool, where string, want []string) {
	t.Helper()

	wanted := make(map[string]bool, len(want))
	for _, name := range want {
		wanted[name] = true
	}
	found := map[string]bool{}
	for _, tool := range offered {
		if !wanted[tool.Name] {
			t.Errorf("%s is served on %s", tool.Name, where)
		}
		found[tool.Name] = true
	}
	for _, name := range want {
		if !found[name] {
			t.Errorf("%s is not served on %s", name, where)
		}
	}
}
