package container_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	vaultPlugin = "md.test.vault"
	cardPlugin  = "md.test.cards"
	cardKey     = cardPlugin + "/card"
	mcpPlugin   = "md.test.mcp"
)

// bearerTransport presents the token on every request.
type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(req)
}

// toolPlugin is a plugin that lists tools and answers every call with its name, its arguments and
// the card of its context.
type toolPlugin struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
	tools []*v1.PluginTool

	mu      sync.Mutex
	lists   int
	request []*v1.CallToolRequest
}

func (p *toolPlugin) ListTools(
	context.Context, *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	p.mu.Lock()
	p.lists++
	p.mu.Unlock()
	return connect.NewResponse(&v1.ListToolsResponse{Tools: p.tools}), nil
}

func (p *toolPlugin) CallTool(
	_ context.Context, req *connect.Request[v1.CallToolRequest],
) (*connect.Response[v1.CallToolResponse], error) {
	p.mu.Lock()
	p.request = append(p.request, req.Msg)
	p.mu.Unlock()
	text := fmt.Sprintf("%s %s %s", req.Msg.GetName(), req.Msg.GetArguments(), req.Msg.GetContext()[cardKey])
	return connect.NewResponse(&v1.CallToolResponse{Result: &v1.ToolResult{Text: text}}), nil
}

func (p *toolPlugin) getRequests() []*v1.CallToolRequest {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.request)
}

func (p *toolPlugin) countLists() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lists
}

func newPluginTool(name string, effect v1.ToolEffect) *v1.PluginTool {
	return &v1.PluginTool{
		Name: name, Title: name, Effect: effect,
		InputSchema: []byte(`{"type":"object","properties":{"path":{"type":"string"}}}`),
	}
}

// testPlugin is one plugin of a test host: its manifest and the services its backend serves.
type testPlugin struct {
	id           string
	capabilities []string
	permissions  []string
	mount        func(mux *http.ServeMux)
}

func hostOver(t *testing.T, plugins ...testPlugin) *container.PluginRoles {
	t.Helper()
	tmp := t.TempDir()
	ports := map[string]int{}
	for _, one := range plugins {
		dir := filepath.Join(tmp, one.id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		capabilities, err := json.Marshal(one.capabilities)
		if err != nil {
			t.Fatal(err)
		}
		permissions, err := json.Marshal(one.permissions)
		if err != nil {
			t.Fatal(err)
		}
		manifest := fmt.Sprintf(`{
			"id": %q, "name": %q, "version": "1.0.0",
			"capabilities": %s, "permissions": %s,
			"targets": ["desktop.editor"],
			"services": ["numen.v1.PluginToolService"], "backend": {"entrypoint": "bin/x"}
		}`, one.id, one.id, capabilities, permissions)
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		mux := http.NewServeMux()
		wiretest.HandlePluginLifecycle(mux)
		one.mount(mux)
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)
		addr, isTCP := server.Listener.Addr().(*net.TCPAddr)
		if !isTCP {
			t.Fatalf("expected *net.TCPAddr, got %T", server.Listener.Addr())
		}
		ports[one.id] = addr.Port
	}

	host := container.NewPluginRoles(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}
	host.Processes.SetLauncher(func(ctx context.Context, p manifest.Discovered, _ int, _ string) (*os.Process, io.ReadCloser, error) {
		port, isServed := ports[p.Manifest.ID]
		if !isServed {
			return nil, nil, errors.New("this plugin cannot be started")
		}
		cmd := exec.CommandContext(context.WithoutCancel(ctx), "sleep", "10")
		if err := cmd.Start(); err != nil {
			return nil, nil, err
		}
		t.Cleanup(func() { wiretest.Kill(t, cmd) })
		return cmd.Process, io.NopCloser(strings.NewReader(fmt.Sprintf("NUMEN_READY port=%d\n", port))), nil
	})
	return host
}

func toolsAt(id string, permissions []string, tools *toolPlugin) testPlugin {
	return testPlugin{
		id: id, capabilities: []string{"tools"}, permissions: permissions,
		mount: func(mux *http.ServeMux) { mux.Handle(numenv1connect.NewPluginToolServiceHandler(tools)) },
	}
}

func newCaller(permissions ...string) testPlugin {
	return testPlugin{id: mcpPlugin, permissions: permissions, mount: func(*http.ServeMux) {}}
}

// shownVault is a host that shows one vault and links another.
type shownVault struct {
	vaultapi.Switch
	vault domain.Vault
	other domain.Vault
}

func (s shownVault) GetVault() (domain.Vault, bool) { return s.vault, true }

func (s shownVault) With(id domain.VaultID) (*vaultapi.Access, error) {
	switch id {
	case s.vault.ID:
		return &vaultapi.Access{Switch: s}, nil
	case s.other.ID:
		return &vaultapi.Access{Switch: shownVault{vault: s.other, other: s.vault}}, nil
	}
	return nil, os.ErrNotExist
}

// A plugin lists the tools of the tools plugins on the host, and a call runs with the context it
// carries in the vault the host shows. A tool whose effect it holds no permission of is neither
// listed nor run.
func TestAPluginListsAndCallsTheToolsOfTheOthersThroughTheHost(t *testing.T) {
	vault := &toolPlugin{tools: []*v1.PluginTool{
		newPluginTool("note_read", v1.ToolEffect_TOOL_EFFECT_READ_ONLY),
		newPluginTool("note_write", v1.ToolEffect_TOOL_EFFECT_READ_WRITE),
	}}
	host := hostOver(t, toolsAt(vaultPlugin, []string{"vault:read", "vault:write"}, vault), newCaller("vault:read"))
	client := openToolsOfHost(t, host)

	if names := getToolNames(listToolsOfHost(t, client)); !slices.Equal(names, []string{"note_read"}) {
		t.Errorf("the host lists %v", names)
	}
	res, err := client.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{
		Name: "note_read", Arguments: []byte(`{"path":"Garden.md"}`), Context: map[string]string{cardKey: "c-1"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if text := res.Msg.GetResult().GetText(); text != `note_read {"path":"Garden.md"} c-1` {
		t.Errorf("the call answered %q", text)
	}
	requests := vault.getRequests()
	if len(requests) != 1 || requests[0].GetVaultId() != "v-orchard" || !maps.Equal(requests[0].GetContext(), map[string]string{cardKey: "c-1"}) {
		t.Errorf("the plugin was called with %v", requests)
	}
	denial, err := client.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{Name: "note_write"}))
	if err != nil {
		t.Fatal(err)
	}
	if denial.Msg.GetError() != v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL || len(vault.getRequests()) != 1 {
		t.Errorf("a tool the caller may not run answered %v", denial.Msg)
	}
}

// A call that names a vault runs in it: a task hands the tools the vault it works.
func TestACallRunsInTheVaultItNames(t *testing.T) {
	cards := &toolPlugin{tools: []*v1.PluginTool{newPluginTool("card_current_get", v1.ToolEffect_TOOL_EFFECT_READ_ONLY)}}
	host := hostOver(t, toolsAt(cardPlugin, []string{"vault:read"}, cards), newCaller("vault:read"))
	client := openToolsOfHost(t, host)

	if _, err := client.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{
		Name: "card_current_get", Arguments: []byte(`{}`), Context: map[string]string{cardKey: "c-9"}, VaultId: "v-garden",
	})); err != nil {
		t.Fatal(err)
	}

	if requests := cards.getRequests(); len(requests) != 1 || requests[0].GetVaultId() != "v-garden" || requests[0].GetContext()[cardKey] != "c-9" {
		t.Errorf("the plugin was called with %v", requests)
	}
}

// A tool is listed only when its plugin holds the permission for its effect and names an effect;
// a plugin that does not declare the tools capability is not queried.
func TestOnlyTheToolsThePermissionsAllowAreListed(t *testing.T) {
	reader := &toolPlugin{tools: []*v1.PluginTool{
		newPluginTool("note_read", v1.ToolEffect_TOOL_EFFECT_READ_ONLY),
		newPluginTool("note_write", v1.ToolEffect_TOOL_EFFECT_READ_WRITE),
		newPluginTool("note_guess", v1.ToolEffect_TOOL_EFFECT_UNSPECIFIED),
	}}
	undeclared := &toolPlugin{tools: []*v1.PluginTool{newPluginTool("card_current_get", v1.ToolEffect_TOOL_EFFECT_READ_ONLY)}}
	host := hostOver(t,
		toolsAt(vaultPlugin, []string{"vault:read"}, reader),
		testPlugin{
			id: cardPlugin, permissions: []string{"vault:read"},
			mount: func(mux *http.ServeMux) { mux.Handle(numenv1connect.NewPluginToolServiceHandler(undeclared)) },
		},
		newCaller("vault:read", "vault:write"),
	)

	if names := getToolNames(listToolsOfHost(t, openToolsOfHost(t, host))); !slices.Equal(names, []string{"note_read"}) {
		t.Errorf("the host lists %v", names)
	}
	if undeclared.countLists() != 0 {
		t.Error("a plugin that does not declare the tools capability was asked for its tools")
	}
}
