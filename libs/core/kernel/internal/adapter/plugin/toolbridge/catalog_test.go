package toolbridge_test

import (
	"context"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

const (
	gardenPlugin  = "md.test.garden"
	orchardPlugin = "md.test.orchard"
	callerPlugin  = "md.test.caller"
)

// toolPlugin answers ListTools with its tools and every call with the tool's name.
type toolPlugin struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
	tools []*v1.PluginTool
}

func (p *toolPlugin) ListTools(
	context.Context, *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	return connect.NewResponse(&v1.ListToolsResponse{Tools: p.tools}), nil
}

func (p *toolPlugin) CallTool(
	_ context.Context, req *connect.Request[v1.CallToolRequest],
) (*connect.Response[v1.CallToolResponse], error) {
	return connect.NewResponse(&v1.CallToolResponse{Result: &v1.ToolResult{Text: req.Msg.GetName()}}), nil
}

// pluginHost serves each plugin from a test server and records every start, and apart the starts for a
// question.
type pluginHost struct {
	mu        sync.Mutex
	infos     []plugin.Info
	disabled  map[string]bool
	ports     map[string]int
	starts    []string
	questions []string
	holders   []plugin.Holder
}

func newPluginHost(t *testing.T, plugins map[string]*toolPlugin) *pluginHost {
	t.Helper()
	host := &pluginHost{disabled: map[string]bool{}, ports: map[string]int{}}
	ids := slices.Sorted(maps.Keys(plugins))
	for _, id := range ids {
		mux := http.NewServeMux()
		mux.Handle(numenv1connect.NewPluginToolServiceHandler(plugins[id]))
		server := httptest.NewServer(mux)
		t.Cleanup(server.Close)
		addr, isTCP := server.Listener.Addr().(*net.TCPAddr)
		if !isTCP {
			t.Fatalf("expected *net.TCPAddr, got %T", server.Listener.Addr())
		}
		host.ports[id] = addr.Port
		host.infos = append(host.infos, plugin.Info{
			ID: id, Version: "1.0.0", Capabilities: []string{"tools"}, Permissions: []string{"vault:read"},
		})
	}
	host.infos = append(host.infos, plugin.Info{ID: callerPlugin, Version: "1.0.0", Permissions: []string{"vault:read"}})
	return host
}

func (h *pluginHost) EnsureActive(_ context.Context, pluginID string, holder plugin.Holder) (transport.Endpoint, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.starts = append(h.starts, pluginID)
	h.holders = append(h.holders, holder)
	return transport.Endpoint{Host: "127.0.0.1", Port: h.ports[pluginID], Token: "t-" + pluginID}, nil
}

func (h *pluginHost) Release(string, plugin.Holder) {}

func (h *pluginHost) EnsureActiveForQuestion(ctx context.Context, pluginID string, holder plugin.Holder) (transport.Endpoint, error) {
	h.mu.Lock()
	h.questions = append(h.questions, pluginID)
	h.mu.Unlock()
	return h.EnsureActive(ctx, pluginID, holder)
}

func (h *pluginHost) ReleaseQuestion(string, plugin.Holder) {}

func (h *pluginHost) GetPluginInfos() []plugin.Info {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.infos)
}

func (h *pluginHost) IsEnabled(pluginID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return !h.disabled[pluginID]
}

func (h *pluginHost) setVersion(pluginID, version string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.infos {
		if h.infos[i].ID == pluginID {
			h.infos[i].Version = version
		}
	}
}

func (h *pluginHost) disable(pluginID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.disabled[pluginID] = true
}

func (h *pluginHost) takeStarts() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	starts := h.starts
	h.starts = nil
	return starts
}

func (h *pluginHost) takeQuestions() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	questions := h.questions
	h.questions = nil
	return questions
}

func newTwoPlugins(t *testing.T) *pluginHost {
	t.Helper()
	return newPluginHost(t, map[string]*toolPlugin{
		gardenPlugin:  {tools: []*v1.PluginTool{{Name: "bed_list", Effect: v1.ToolEffect_TOOL_EFFECT_READ_ONLY}}},
		orchardPlugin: {tools: []*v1.PluginTool{{Name: "tree_count", Effect: v1.ToolEffect_TOOL_EFFECT_READ_ONLY}}},
	})
}

func getNames(tools []*v1.PluginTool) []string {
	names := make([]string, 0, len(tools))
	for _, one := range tools {
		names = append(names, one.GetName())
	}
	return names
}

func filesIn(t *testing.T, dir string) *storage.Storage {
	t.Helper()
	files := storage.New(func(err error) { t.Errorf("the storage reported %v", err) }, func(string) string { return "1.0.0" })
	files.SetStorageRoots(dir, "")
	return files
}

func TestTheFirstListingStartsEveryToolsPluginAndTheSecondNone(t *testing.T) {
	host := newTwoPlugins(t)
	catalog := newCatalog(host, filesIn(t, ""))

	if names := getNames(catalog.ListTools(t.Context(), callerPlugin)); !slices.Equal(names, []string{"bed_list", "tree_count"}) {
		t.Fatalf("the first listing gave %v", names)
	}
	if starts := host.takeStarts(); !slices.Equal(starts, []string{gardenPlugin, orchardPlugin}) {
		t.Fatalf("the first listing started %v", starts)
	}
	if names := getNames(catalog.ListTools(t.Context(), callerPlugin)); !slices.Equal(names, []string{"bed_list", "tree_count"}) {
		t.Fatalf("the second listing gave %v", names)
	}
	if starts := host.takeStarts(); len(starts) != 0 {
		t.Errorf("the second listing started %v", starts)
	}
}

func TestANewVersionOfAPluginIsListedAgain(t *testing.T) {
	host := newTwoPlugins(t)
	catalog := newCatalog(host, filesIn(t, ""))
	catalog.ListTools(t.Context(), callerPlugin)
	host.takeStarts()

	host.setVersion(orchardPlugin, "1.1.0")
	catalog.ListTools(t.Context(), callerPlugin)

	if starts := host.takeStarts(); !slices.Equal(starts, []string{orchardPlugin}) {
		t.Errorf("the listing after the update started %v", starts)
	}
}

func TestACallStartsOnlyThePluginOfTheTool(t *testing.T) {
	host := newTwoPlugins(t)
	catalog := newCatalog(host, filesIn(t, ""))
	catalog.ListTools(t.Context(), callerPlugin)
	host.takeStarts()

	res, err := catalog.CallTool(t.Context(), callerPlugin, &v1.CallToolRequest{Name: "tree_count"})
	if err != nil {
		t.Fatal(err)
	}

	if res.GetResult().GetText() != "tree_count" {
		t.Errorf("the call answered %v", res)
	}
	if starts := host.takeStarts(); !slices.Equal(starts, []string{orchardPlugin}) {
		t.Errorf("the call started %v", starts)
	}
}

func TestTheListingAsksThePluginsAQuestionAndACallUsesThePlugin(t *testing.T) {
	host := newTwoPlugins(t)
	catalog := newCatalog(host, filesIn(t, ""))

	catalog.ListTools(t.Context(), callerPlugin)
	if questions := host.takeQuestions(); !slices.Equal(questions, []string{gardenPlugin, orchardPlugin}) {
		t.Errorf("the listing asked %v a question", questions)
	}

	if _, err := catalog.CallTool(t.Context(), callerPlugin, &v1.CallToolRequest{Name: "tree_count"}); err != nil {
		t.Fatal(err)
	}
	if questions := host.takeQuestions(); len(questions) != 0 {
		t.Errorf("the call held %v for a question", questions)
	}
}

func TestADisabledPluginsToolsAreNotListed(t *testing.T) {
	host := newTwoPlugins(t)
	catalog := newCatalog(host, filesIn(t, ""))
	catalog.ListTools(t.Context(), callerPlugin)

	host.disable(gardenPlugin)

	if names := getNames(catalog.ListTools(t.Context(), callerPlugin)); !slices.Equal(names, []string{"tree_count"}) {
		t.Errorf("the listing after disabling gave %v", names)
	}
	res, err := catalog.CallTool(t.Context(), callerPlugin, &v1.CallToolRequest{Name: "bed_list"})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetError() != v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL {
		t.Errorf("a call to a tool of a disabled plugin answered %v", res)
	}
}

func TestTheListingsOnDiskServeANewCatalog(t *testing.T) {
	dir := t.TempDir()
	newCatalog(newTwoPlugins(t), filesIn(t, dir)).ListTools(t.Context(), callerPlugin)

	host := newTwoPlugins(t)
	names := getNames(newCatalog(host, filesIn(t, dir)).ListTools(t.Context(), callerPlugin))

	if !slices.Equal(names, []string{"bed_list", "tree_count"}) {
		t.Errorf("the new catalog lists %v", names)
	}
	if starts := host.takeStarts(); len(starts) != 0 {
		t.Errorf("the new catalog started %v", starts)
	}
}

func TestACallForAConnectionHoldsThePluginOfTheToolInTheNameOfTheConnection(t *testing.T) {
	host := newTwoPlugins(t)
	catalog := newCatalog(host, filesIn(t, ""))
	catalog.ListTools(t.Context(), callerPlugin)
	host.mu.Lock()
	host.holders = nil
	host.mu.Unlock()
	ctx := plugin.WithCaller(t.Context(), plugin.Caller{PluginID: callerPlugin, ConnectionID: "conn-keeper"})

	if _, err := catalog.CallTool(ctx, callerPlugin, &v1.CallToolRequest{Name: "tree_count", Arguments: []byte(`{}`)}); err != nil {
		t.Fatal(err)
	}

	host.mu.Lock()
	defer host.mu.Unlock()
	want := []plugin.Holder{{PluginID: callerPlugin, ConnectionID: "conn-keeper", Why: "call tool tree_count"}}
	if !slices.Equal(host.holders, want) {
		t.Errorf("the call held the plugin for %+v, want %+v", host.holders, want)
	}
}
