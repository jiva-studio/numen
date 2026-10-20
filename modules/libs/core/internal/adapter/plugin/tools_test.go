package plugin

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// toolPlugin serves PluginToolService with the tools it is given and keeps the calls it was made.
type toolPlugin struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
	mu     sync.Mutex
	tools  []*v1.PluginTool
	reply  *v1.CallToolResponse
	called []*v1.CallToolRequest
}

func (p *toolPlugin) ListTools(
	context.Context, *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	return connect.NewResponse(&v1.ListToolsResponse{Tools: p.tools}), nil
}

func (p *toolPlugin) CallTool(
	_ context.Context, req *connect.Request[v1.CallToolRequest],
) (*connect.Response[v1.CallToolResponse], error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.called = append(p.called, req.Msg)
	return connect.NewResponse(p.reply), nil
}

// toolsHost runs the plugins it is given, each on a server of its own, and holds each to the
// permissions its entry names.
type toolsHost struct {
	ids       []string
	endpoints map[string]port.Endpoint
	holds     map[string][]string
	acquired  int
	released  int
}

func (h *toolsHost) ResolveCapability(string) (port.DiscoveredPlugin, error) {
	return port.DiscoveredPlugin{}, &port.UnavailableError{Reason: port.PluginNotInstalled}
}

func (h *toolsHost) ResolveService(string) (port.DiscoveredPlugin, bool) {
	return port.DiscoveredPlugin{}, false
}

func (h *toolsHost) ResolveCapabilities(capability string) []port.DiscoveredPlugin {
	if capability != CapabilityTools {
		return nil
	}
	found := make([]port.DiscoveredPlugin, 0, len(h.ids))
	for _, id := range h.ids {
		found = append(found, port.DiscoveredPlugin{Manifest: port.PluginManifest{ID: id}})
	}
	return found
}

func (h *toolsHost) GetPlugin(string) (port.DiscoveredPlugin, bool) {
	return port.DiscoveredPlugin{}, false
}

func (h *toolsHost) EnsureActive(_ context.Context, id string) (port.Endpoint, error) {
	endpoint, running := h.endpoints[id]
	if !running {
		return port.Endpoint{}, errors.New("the plugin is down")
	}
	h.acquired++
	return endpoint, nil
}

func (h *toolsHost) Release(string) { h.released++ }

func (h *toolsHost) CheckPluginPermission(id, permission string) error {
	for _, held := range h.holds[id] {
		if held == permission {
			return nil
		}
	}
	return port.ErrPermissionDenied
}

// serveToolPlugin serves the plugin and returns where it is.
func serveToolPlugin(t *testing.T, plugin *toolPlugin) port.Endpoint {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewPluginToolServiceHandler(plugin))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	host, rawPort, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.Atoi(rawPort)
	if err != nil {
		t.Fatal(err)
	}
	return port.Endpoint{Host: host, Port: number}
}

func wireTool(name string, effect v1.ToolEffect) *v1.PluginTool {
	return &v1.PluginTool{
		Name:        name,
		Description: name + " does it",
		InputSchema: []byte(`{"type":"object"}`),
		Effect:      effect,
	}
}

func TestToolsAreListedFromEveryPluginThatOffersThemInTheOrderOfTheirIds(t *testing.T) {
	first := &toolPlugin{tools: []*v1.PluginTool{wireTool("a_read", v1.ToolEffect_TOOL_EFFECT_READS)}}
	second := &toolPlugin{tools: []*v1.PluginTool{{
		Name: "b_drop", InputSchema: []byte(`{"type":"object"}`), Effect: v1.ToolEffect_TOOL_EFFECT_WRITES,
		Servers: []v1.ToolServer{v1.ToolServer_TOOL_SERVER_FULL, v1.ToolServer_TOOL_SERVER_REVIEW},
		Step:    v1.ToolStep_TOOL_STEP_REMOVE,
	}}}
	host := &toolsHost{
		ids: []string{"md.numen.a", "md.numen.b"},
		endpoints: map[string]port.Endpoint{
			"md.numen.a": serveToolPlugin(t, first),
			"md.numen.b": serveToolPlugin(t, second),
		},
		holds: map[string][]string{
			"md.numen.a": {port.PermissionVaultRead},
			"md.numen.b": {port.PermissionVaultRead, port.PermissionVaultWrite},
		},
	}

	tools := OpenToolsBridge(host).ListTools(t.Context())

	if len(tools) != 2 || tools[0].PluginID != "md.numen.a" || tools[1].PluginID != "md.numen.b" {
		t.Fatalf("the tools were %+v", tools)
	}
	drop := tools[1]
	if drop.Name != "b_drop" || drop.Effect != port.ToolWrites || drop.Step != port.ToolStepRemove ||
		len(drop.Servers) != 2 || !drop.IsOn(port.ToolServerReview) || drop.IsOn(port.ToolServerReading) {
		t.Errorf("the second plugin's tool came out as %+v", drop)
	}
	if host.acquired != host.released {
		t.Errorf("acquired %d references, gave back %d", host.acquired, host.released)
	}
}

func TestAToolIsNotOfferedByAPluginThatLacksThePermissionItsEffectNeeds(t *testing.T) {
	plugin := &toolPlugin{tools: []*v1.PluginTool{
		wireTool("reads", v1.ToolEffect_TOOL_EFFECT_READS),
		wireTool("writes", v1.ToolEffect_TOOL_EFFECT_WRITES),
		wireTool("nothing", v1.ToolEffect_TOOL_EFFECT_UNSPECIFIED),
		{Effect: v1.ToolEffect_TOOL_EFFECT_READS},
	}}
	host := &toolsHost{
		ids:       []string{"md.numen.a"},
		endpoints: map[string]port.Endpoint{"md.numen.a": serveToolPlugin(t, plugin)},
		holds:     map[string][]string{"md.numen.a": {port.PermissionVaultRead}},
	}

	tools := OpenToolsBridge(host).ListTools(t.Context())

	if len(tools) != 1 || tools[0].Name != "reads" {
		t.Errorf("the tools were %+v", tools)
	}
}

func TestAPluginThatCannotBeStartedIsLeftOutAndTheRestAreListed(t *testing.T) {
	plugin := &toolPlugin{tools: []*v1.PluginTool{wireTool("reads", v1.ToolEffect_TOOL_EFFECT_READS)}}
	host := &toolsHost{
		ids:       []string{"md.numen.a", "md.numen.b"},
		endpoints: map[string]port.Endpoint{"md.numen.b": serveToolPlugin(t, plugin)},
		holds:     map[string][]string{"md.numen.b": {port.PermissionVaultRead}},
	}

	tools := OpenToolsBridge(host).ListTools(t.Context())

	if len(tools) != 1 || tools[0].PluginID != "md.numen.b" {
		t.Errorf("the tools were %+v", tools)
	}
}

func TestACallReachesThePluginThatListedTheToolAndReturnsWhatItAnswered(t *testing.T) {
	plugin := &toolPlugin{
		tools: []*v1.PluginTool{wireTool("reads", v1.ToolEffect_TOOL_EFFECT_READS)},
		reply: &v1.CallToolResponse{Result: &v1.ToolResult{Text: "done", Structured: []byte(`{"n":1}`), IsRefusal: true}},
	}
	host := &toolsHost{
		ids:       []string{"md.numen.a"},
		endpoints: map[string]port.Endpoint{"md.numen.a": serveToolPlugin(t, plugin)},
		holds:     map[string][]string{"md.numen.a": {port.PermissionVaultRead}},
	}
	bridge := OpenToolsBridge(host)
	bridge.ListTools(t.Context())

	result, err := bridge.CallTool(t.Context(), "md.numen.a", "reads", []byte(`{"x":1}`), map[string]string{"mark": "k7"})
	if err != nil {
		t.Fatal(err)
	}

	if result.Text != "done" || string(result.Structured) != `{"n":1}` || !result.IsRefusal {
		t.Errorf("the result was %+v", result)
	}
	if len(plugin.called) != 1 || plugin.called[0].GetName() != "reads" || string(plugin.called[0].GetArguments()) != `{"x":1}` ||
		plugin.called[0].GetAsked()["mark"] != "k7" {
		t.Errorf("the plugin was asked %v", plugin.called)
	}
}

func TestACallForAToolNoPluginListedIsRefusedWithoutReachingAPlugin(t *testing.T) {
	plugin := &toolPlugin{tools: []*v1.PluginTool{wireTool("reads", v1.ToolEffect_TOOL_EFFECT_READS)}}
	host := &toolsHost{
		ids:       []string{"md.numen.a"},
		endpoints: map[string]port.Endpoint{"md.numen.a": serveToolPlugin(t, plugin)},
		holds:     map[string][]string{"md.numen.a": {port.PermissionVaultRead}},
	}
	bridge := OpenToolsBridge(host)

	_, beforeListing := bridge.CallTool(t.Context(), "md.numen.a", "reads", nil, nil)
	bridge.ListTools(t.Context())
	_, wrongPlugin := bridge.CallTool(t.Context(), "md.numen.b", "reads", nil, nil)
	_, wrongName := bridge.CallTool(t.Context(), "md.numen.a", "other", nil, nil)

	for name, err := range map[string]error{"before": beforeListing, "plugin": wrongPlugin, "name": wrongName} {
		if !errors.Is(err, ErrUnknownTool) {
			t.Errorf("a call %s listing is %v, want ErrUnknownTool", name, err)
		}
	}
	if len(plugin.called) != 0 {
		t.Errorf("the plugin was reached: %v", plugin.called)
	}
}

func TestACallIsRefusedOnceThePluginNoLongerHoldsThePermission(t *testing.T) {
	plugin := &toolPlugin{tools: []*v1.PluginTool{wireTool("writes", v1.ToolEffect_TOOL_EFFECT_WRITES)}}
	host := &toolsHost{
		ids:       []string{"md.numen.a"},
		endpoints: map[string]port.Endpoint{"md.numen.a": serveToolPlugin(t, plugin)},
		holds:     map[string][]string{"md.numen.a": {port.PermissionVaultWrite}},
	}
	bridge := OpenToolsBridge(host)
	bridge.ListTools(t.Context())
	host.holds["md.numen.a"] = nil

	_, err := bridge.CallTool(t.Context(), "md.numen.a", "writes", nil, nil)

	if !errors.Is(err, port.ErrPermissionDenied) {
		t.Errorf("the call was answered with %v", err)
	}
	if len(plugin.called) != 0 {
		t.Errorf("the plugin was reached: %v", plugin.called)
	}
}

func TestACallTheToolRefusesAsMalformedOrUnknownIsAnErrorOfTheBridge(t *testing.T) {
	for code, want := range map[v1.ToolErrorCode]error{
		v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL:  ErrUnknownTool,
		v1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS: ErrBadToolArguments,
	} {
		plugin := &toolPlugin{
			tools: []*v1.PluginTool{wireTool("reads", v1.ToolEffect_TOOL_EFFECT_READS)},
			reply: &v1.CallToolResponse{Error: code},
		}
		host := &toolsHost{
			ids:       []string{"md.numen.a"},
			endpoints: map[string]port.Endpoint{"md.numen.a": serveToolPlugin(t, plugin)},
			holds:     map[string][]string{"md.numen.a": {port.PermissionVaultRead}},
		}
		bridge := OpenToolsBridge(host)
		bridge.ListTools(t.Context())

		if _, err := bridge.CallTool(t.Context(), "md.numen.a", "reads", nil, nil); !errors.Is(err, want) {
			t.Errorf("code %v was answered with %v, want %v", code, err, want)
		}
	}
}
