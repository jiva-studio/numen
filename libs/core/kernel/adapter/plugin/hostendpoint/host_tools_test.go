package hostendpoint

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// toolCall records one call to run a tool.
type toolCall struct {
	callerID string
	name     string
	context  map[string]string
	vault    domain.VaultID
	caller   plugin.Caller
}

// recordedTools lists one tool and answers every call with the vault it was run in.
type recordedTools struct {
	listedFor []string
	calls     []toolCall
}

func (r *recordedTools) ListTools(_ context.Context, callerID string) []*v1.PluginTool {
	r.listedFor = append(r.listedFor, callerID)
	return []*v1.PluginTool{{Name: "note_read", Effect: v1.ToolEffect_TOOL_EFFECT_READ_ONLY}}
}

func (r *recordedTools) CallTool(ctx context.Context, callerID string, call *v1.CallToolRequest) (*v1.CallToolResponse, error) {
	caller, _ := plugin.GetCaller(ctx)
	r.calls = append(r.calls, toolCall{
		callerID: callerID, name: call.GetName(), context: call.GetContext(), vault: domain.VaultID(call.GetVaultId()), caller: caller,
	})
	return &v1.CallToolResponse{Result: &v1.ToolResult{Text: call.GetVaultId()}}, nil
}

func newToolClient(t *testing.T, fakeCallers *callers, token string) numenv1connect.PluginToolServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginToolServiceHandler(&toolEndpoint{endpoint: &Endpoint{callers: fakeCallers.asCallers()}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewPluginToolServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor(token)))
}

func newTwoVaults(tools *recordedTools) *callers {
	vault := newMemoryVault(nil)
	vault.others = map[domain.VaultID]*memoryVault{"v2": {vault: &domain.Vault{ID: "v2", Name: "Biology"}}}
	fakeCallers := newCallers()
	fakeCallers.tools, fakeCallers.vault = tools, vaultOf(vault)
	return fakeCallers
}

// A plugin lists the tools it is offered under its own id, and a call that names no vault runs in
// the vault the host shows, with the context of the call.
func TestAPluginCallsTheToolsInTheVaultTheHostShows(t *testing.T) {
	tools := &recordedTools{}
	client := newToolClient(t, newTwoVaults(tools), "token-a")

	listed, err := client.ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := listed.Msg.GetTools(); len(got) != 1 || got[0].GetName() != "note_read" {
		t.Errorf("listed %v", got)
	}
	res, err := client.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{
		Name: "note_read", Context: map[string]string{"md.numen.a/card": "c-1"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetResult().GetText() != "v1" {
		t.Errorf("the call answered %v", res.Msg)
	}
	if len(tools.calls) != 1 || tools.calls[0].callerID != "md.numen.a" || tools.calls[0].name != "note_read" ||
		tools.calls[0].vault != "v1" || !maps.Equal(tools.calls[0].context, map[string]string{"md.numen.a/card": "c-1"}) {
		t.Errorf("the tools ran %+v", tools.calls)
	}
	if len(tools.listedFor) != 1 || tools.listedFor[0] != "md.numen.a" {
		t.Errorf("the tools were listed for %v", tools.listedFor)
	}
}

// A call that names a vault by the Numen-Vault header runs in it.
func TestAPluginCallsTheToolsInTheVaultItsHeaderNames(t *testing.T) {
	tools := &recordedTools{}
	req := connect.NewRequest(&v1.CallToolRequest{Name: "note_read"})
	req.Header().Set(vaultapi.Header, "v2")
	if _, err := newToolClient(t, newTwoVaults(tools), "token-a").CallTool(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	if len(tools.calls) != 1 || tools.calls[0].vault != "v2" {
		t.Errorf("the tools ran %+v", tools.calls)
	}
}

// A call that names a vault in the request runs in it, whatever the header names.
func TestAPluginCallsTheToolsInTheVaultTheRequestNames(t *testing.T) {
	tools := &recordedTools{}
	req := connect.NewRequest(&v1.CallToolRequest{Name: "note_read", VaultId: "v2"})
	req.Header().Set(vaultapi.Header, "v1")
	if _, err := newToolClient(t, newTwoVaults(tools), "token-a").CallTool(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	if len(tools.calls) != 1 || tools.calls[0].vault != "v2" {
		t.Errorf("the tools ran %+v", tools.calls)
	}
}

// A call that names a vault the host does not know runs nowhere.
func TestACallInAVaultTheHostDoesNotKnowIsNotFound(t *testing.T) {
	tools := &recordedTools{}
	_, err := newToolClient(t, newTwoVaults(tools), "token-a").CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{
		Name: "note_read", VaultId: "v-unknown",
	}))
	if got := readCode(t, err); got != connect.CodeNotFound {
		t.Errorf("code = %v, want not found", got)
	}
	if len(tools.calls) != 0 {
		t.Errorf("the tools ran %+v", tools.calls)
	}
}

// A plugin without vault:read is rejected, and an unknown caller is not served.
func TestTheToolsRefuseACallerWithoutVaultRead(t *testing.T) {
	tools := &recordedTools{}
	fakeCallers := newCallers()
	fakeCallers.tools = tools

	_, err := newToolClient(t, fakeCallers, "token-b").ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if got := readCode(t, err); got != connect.CodePermissionDenied {
		t.Errorf("code = %v, want permission denied", got)
	}
	_, err = newToolClient(t, fakeCallers, "token-x").CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{Name: "note_read"}))
	if got := readCode(t, err); got != connect.CodeUnauthenticated {
		t.Errorf("code = %v, want unauthenticated", got)
	}
	if len(tools.listedFor) != 0 || len(tools.calls) != 0 {
		t.Errorf("the tools were reached: %v, %v", tools.listedFor, tools.calls)
	}
}

// A call an agent makes for a connection runs in the name of the connection.
func TestAToolCallForAConnectionRunsInTheNameOfTheConnection(t *testing.T) {
	tools := &recordedTools{}
	req := connect.NewRequest(&v1.CallToolRequest{Name: "note_read", ConnectionId: "conn-keeper"})
	if _, err := newToolClient(t, newTwoVaults(tools), "token-a").CallTool(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	want := plugin.Caller{PluginID: "md.numen.a", ConnectionID: "conn-keeper"}
	if len(tools.calls) != 1 || tools.calls[0].caller != want {
		t.Errorf("the tools ran %+v, want the caller %+v", tools.calls, want)
	}
}
