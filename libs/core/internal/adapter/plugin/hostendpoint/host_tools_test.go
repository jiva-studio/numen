package hostendpoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
)

// toolCall records one call to run a tool.
type toolCall struct {
	callerID string
	name     string
	vault    domain.VaultID
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

func (r *recordedTools) CallTool(
	_ context.Context, callerID, name string, _ []byte, vault domain.VaultID,
) (*v1.CallToolResponse, error) {
	r.calls = append(r.calls, toolCall{callerID: callerID, name: name, vault: vault})
	return &v1.CallToolResponse{Result: &v1.ToolResult{Text: string(vault)}}, nil
}

// newToolClient serves the tools of the callers and returns a client calling them with the token.
func newToolClient(t *testing.T, fakeCallers *callers, token string) numenv1connect.PluginToolServiceClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginToolServiceHandler(&toolEndpoint{endpoint: &Endpoint{callers: fakeCallers}}))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return numenv1connect.NewPluginToolServiceClient(server.Client(), server.URL, connect.WithInterceptors(transport.AuthInterceptor(token)))
}

// A plugin lists the tools it is offered under its own id, and calls them in the vault the host
// shows with no context, whatever context it sends.
func TestAPluginCallsTheToolsInTheVaultTheHostShows(t *testing.T) {
	tools := &recordedTools{}
	fakeCallers := newCallers(nil)
	fakeCallers.tools, fakeCallers.vault = tools, newMemoryVault(nil)
	client := newToolClient(t, fakeCallers, "token-a")

	listed, err := client.ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if got := listed.Msg.GetTools(); len(got) != 1 || got[0].GetName() != "note_read" {
		t.Errorf("listed %v", got)
	}
	res, err := client.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{
		Name: "note_read", Context: map[string]string{"md.numen.a/card": "c-1"}, VaultId: "v-other",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetResult().GetText() != "v1" {
		t.Errorf("the call answered %v", res.Msg)
	}
	if want := (toolCall{callerID: "md.numen.a", name: "note_read", vault: "v1"}); len(tools.calls) != 1 || tools.calls[0] != want {
		t.Errorf("the tools ran %+v, want %+v", tools.calls, want)
	}
	if len(tools.listedFor) != 1 || tools.listedFor[0] != "md.numen.a" {
		t.Errorf("the tools were listed for %v", tools.listedFor)
	}
}

// A call that names a vault by the Numen-Vault header runs in it.
func TestAPluginCallsTheToolsInTheVaultItNames(t *testing.T) {
	tools := &recordedTools{}
	vault := newMemoryVault(nil)
	vault.others = map[domain.VaultID]*memoryVault{"v2": {vault: &domain.Vault{ID: "v2", Name: "Biology"}}}
	fakeCallers := newCallers(nil)
	fakeCallers.tools, fakeCallers.vault = tools, vault

	req := connect.NewRequest(&v1.CallToolRequest{Name: "note_read"})
	req.Header().Set(vaultHeader, "v2")
	if _, err := newToolClient(t, fakeCallers, "token-a").CallTool(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	if len(tools.calls) != 1 || tools.calls[0].vault != "v2" {
		t.Errorf("the tools ran %+v", tools.calls)
	}
}

// A plugin without vault:read is rejected, and an unknown caller is not served.
func TestTheToolsRefuseACallerWithoutVaultRead(t *testing.T) {
	tools := &recordedTools{}
	fakeCallers := newCallers(nil)
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
