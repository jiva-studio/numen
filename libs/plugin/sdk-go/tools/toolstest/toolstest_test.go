package toolstest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools/toolstest"
)

type answering struct {
	listing  tools.Listing
	response *tools.CallResponse
	called   tools.Call
}

func (a *answering) ListTools(context.Context, http.Header) (tools.Listing, error) {
	return a.listing, nil
}

func (a *answering) InvokeTool(_ context.Context, _ http.Header, call tools.Call) (*tools.CallResponse, error) {
	a.called = call
	return a.response, nil
}

func serve(t *testing.T, handler toolstest.ServiceHandler) tools.Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(toolstest.NewServiceHandler(handler))
	web := httptest.NewServer(mux)
	t.Cleanup(web.Close)
	return tools.NewClient(hostcall.Endpoint{BaseURL: web.URL})
}

func TestAToolListedByTheHostReachesTheClientWhole(t *testing.T) {
	want := &tools.ToolInfo{
		Name: "greet", Title: "Greet", Description: "Greets.",
		InputSchema: []byte(`{"type":"object"}`), OutputSchema: []byte(`{"type":"object"}`),
		Effect:       tools.ReadWrite,
		Instructions: []string{"Greet each guest by name with greet.", "Call greet once per guest."},
		Requires:     []string{"vault:manage", "index:read"},
	}
	listing, err := serve(t, &answering{listing: tools.Listing{Tools: []*tools.ToolInfo{want}}}).ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Tools) != 1 || !reflect.DeepEqual(listing.Tools[0], want) {
		t.Errorf("listed %+v, want %+v", listing.Tools, want)
	}
}

func TestEachAnswerOfACallReachesTheClientWhole(t *testing.T) {
	for _, want := range []*tools.CallResponse{
		{Result: &tools.CallResult{Text: "failed", IsError: true}},
		{Result: &tools.CallResult{Structured: []byte(`{"a":1}`)}},
		{Error: tools.ErrorCodeUnknownTool},
		{Error: tools.ErrorCodeBadArguments},
	} {
		host := &answering{response: want}
		call := tools.Call{Name: "greet", Arguments: []byte(`{"name":"Ada"}`), Context: map[string]string{"note": "Guests.md"}, VaultID: "Physics"}
		got, err := serve(t, host).InvokeTool(t.Context(), call)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("answer %+v, want %+v", got, want)
		}
		if !reflect.DeepEqual(host.called, call) {
			t.Errorf("the host was called with %+v, want %+v", host.called, call)
		}
	}
}
