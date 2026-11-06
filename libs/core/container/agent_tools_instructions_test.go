package container_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/agenttoolsbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

const (
	atlasPlugin  = "md.test.atlas"
	zitherPlugin = "md.test.zither"

	atlasOpenSentence   = "Open the atlas with atlas_open before you name a country."
	atlasFoldSentence   = "Fold a map with atlas_fold along its printed creases."
	atlasSharedSentence = "Close the atlas with atlas_close or atlas_fold when the reading ends."
	zitherTuneSentence  = "Tune the zither with zither_tune, one string at a time."
)

// instructedPlugin is a plugin that lists the tools it is given, or fails to list when it is given
// an error.
type instructedPlugin struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
	tools []*v1.PluginTool
	err   error
}

func (p *instructedPlugin) ListTools(
	context.Context, *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	if p.err != nil {
		return nil, connect.NewError(connect.CodeInternal, p.err)
	}
	return connect.NewResponse(&v1.ListToolsResponse{Tools: p.tools}), nil
}

func instructedAt(id string, permissions []string, served *instructedPlugin) testPlugin {
	return testPlugin{
		id: id, capabilities: []string{"tools"}, permissions: permissions,
		mount: func(mux *http.ServeMux) { mux.Handle(numenv1connect.NewPluginToolServiceHandler(served)) },
	}
}

// instructedTool is a tool of a test plugin that carries the instruction sentences given.
func instructedTool(name string, effect v1.ToolEffect, sentences []string, servers ...v1.ToolServer) *v1.PluginTool {
	tool := newPluginTool(name, effect, servers...)
	tool.Instructions = sentences
	return tool
}

func toolsPlugin(tools ...*v1.PluginTool) *instructedPlugin {
	return &instructedPlugin{tools: tools}
}

func readOnlyTool(name string, sentences ...string) *v1.PluginTool {
	return instructedTool(name, v1.ToolEffect_TOOL_EFFECT_READ_ONLY, sentences)
}

// mcpCaller is the plugin that serves the tools of the others outside Numen: it holds vault:read
// only.
func mcpCaller() testPlugin {
	return testPlugin{id: mcpPlugin, permissions: []string{"vault:read"}, mount: func(*http.ServeMux) {}}
}

// openToolsOfHost serves the host endpoint and returns a client calling its PluginToolService as
// the MCP plugin.
func openToolsOfHost(t *testing.T, host *plugin.Host) numenv1connect.PluginToolServiceClient {
	t.Helper()
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	host.SetPluginVault(shownVault{vault: domain.Vault{ID: "v-orchard", Name: "Orchard"}})
	served, err := hostendpoint.Open(t.Context(), hostendpoint.Address, host)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := served.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	caller, err := host.EnsureActive(t.Context(), mcpPlugin)
	if err != nil {
		t.Fatal(err)
	}
	return numenv1connect.NewPluginToolServiceClient(
		&http.Client{Transport: bearerTransport{token: caller.Token}}, fmt.Sprintf("http://127.0.0.1:%d", served.GetPort()))
}

func listToolsOfHost(t *testing.T, client numenv1connect.PluginToolServiceClient) *v1.ListToolsResponse {
	t.Helper()
	listing, err := client.ListTools(t.Context(), connect.NewRequest(&v1.ListToolsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	return listing.Msg
}

func getToolNames(listing *v1.ListToolsResponse) []string {
	names := make([]string, 0, len(listing.GetTools()))
	for _, one := range listing.GetTools() {
		names = append(names, one.GetName())
	}
	return names
}

func getSentencesByTool(listing *v1.ListToolsResponse) map[string][]string {
	byTool := map[string][]string{}
	for _, one := range listing.GetTools() {
		byTool[one.GetName()] = one.GetInstructions()
	}
	return byTool
}

func TestTheHostListsEachToolWithItsOwnSentencesInPluginIdOrder(t *testing.T) {
	host := hostOver(t,
		instructedAt(zitherPlugin, []string{"vault:read"}, toolsPlugin(readOnlyTool("zither_tune", zitherTuneSentence))),
		instructedAt(atlasPlugin, []string{"vault:read"}, toolsPlugin(
			readOnlyTool("atlas_open", atlasOpenSentence),
			readOnlyTool("atlas_fold", atlasFoldSentence, atlasSharedSentence),
			readOnlyTool("atlas_close", atlasSharedSentence),
		)),
		mcpCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if names := getToolNames(listing); !slices.Equal(names, []string{"atlas_open", "atlas_fold", "atlas_close", "zither_tune"}) {
		t.Fatalf("the host lists the tools %v", names)
	}
	want := map[string][]string{
		"atlas_open":  {atlasOpenSentence},
		"atlas_fold":  {atlasFoldSentence, atlasSharedSentence},
		"atlas_close": {atlasSharedSentence},
		"zither_tune": {zitherTuneSentence},
	}
	for name, sentences := range getSentencesByTool(listing) {
		if !slices.Equal(sentences, want[name]) {
			t.Errorf("the host lists %s with the sentences %q, want %q", name, sentences, want[name])
		}
	}
}

// The MCP plugin holds vault:read only, so a read-write tool is not listed to it, and the host
// endpoint offers the full server, so a tool offered on the reading server only is not listed.
func TestNoListedToolCarriesTheSentenceOfAToolLeftOut(t *testing.T) {
	burnSentence := "Burn the old maps with atlas_burn after you copy them."
	lensSentence := "Read the small print with atlas_lens."
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read", "vault:write"}, toolsPlugin(
			readOnlyTool("atlas_open", atlasOpenSentence),
			instructedTool("atlas_burn", v1.ToolEffect_TOOL_EFFECT_READ_WRITE, []string{burnSentence}),
			instructedTool("atlas_lens", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{lensSentence}, v1.ToolServer_TOOL_SERVER_READING),
		)),
		mcpCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if names := getToolNames(listing); !slices.Equal(names, []string{"atlas_open"}) {
		t.Fatalf("the host lists the tools %v", names)
	}
	if got := getSentencesByTool(listing)["atlas_open"]; !slices.Equal(got, []string{atlasOpenSentence}) {
		t.Errorf("the host lists atlas_open with the sentences %q", got)
	}
}

func TestTheHostListsTheToolsOfTheOthersWhenAPluginFailsToList(t *testing.T) {
	broken := toolsPlugin(readOnlyTool("bellows_pump", "Pump the bellows with bellows_pump."))
	broken.err = errors.New("the bellows are torn")
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read"}, toolsPlugin(readOnlyTool("atlas_open", atlasOpenSentence))),
		instructedAt("md.test.bellows", []string{"vault:read"}, broken),
		mcpCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if names := getToolNames(listing); !slices.Equal(names, []string{"atlas_open"}) {
		t.Errorf("the host lists the tools %v", names)
	}
}

func TestDisablingAPluginRemovesItsToolsAndTheirSentences(t *testing.T) {
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read"}, toolsPlugin(readOnlyTool("atlas_open", atlasOpenSentence))),
		instructedAt(zitherPlugin, []string{"vault:read"}, toolsPlugin(readOnlyTool("zither_tune", zitherTuneSentence))),
		mcpCaller(),
	)
	client := openToolsOfHost(t, host)
	if before := getToolNames(listToolsOfHost(t, client)); !slices.Equal(before, []string{"atlas_open", "zither_tune"}) {
		t.Fatalf("before the plugin is disabled the host lists the tools %v", before)
	}

	if err := host.SetPluginEnabled(t.Context(), zitherPlugin, false); err != nil {
		t.Fatal(err)
	}

	if names := getToolNames(listToolsOfHost(t, client)); !slices.Equal(names, []string{"atlas_open"}) {
		t.Errorf("after the plugin is disabled the host lists the tools %v", names)
	}
}

func TestTheToolsOfTheCallerItselfAreNotListed(t *testing.T) {
	caller := instructedAt(mcpPlugin, []string{"vault:read"}, toolsPlugin(readOnlyTool("mirror_look", "Look in the mirror with mirror_look.")))
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read"}, toolsPlugin(readOnlyTool("atlas_open", atlasOpenSentence))),
		caller,
	)

	if names := getToolNames(listToolsOfHost(t, openToolsOfHost(t, host))); !slices.Equal(names, []string{"atlas_open"}) {
		t.Errorf("the host lists the caller the tools %v", names)
	}
}

func TestTheReadingCatalogListsOnlyReadingServerToolsWithTheirSentences(t *testing.T) {
	glanceSentence := "Glance at the atlas index with atlas_glance."
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read"}, toolsPlugin(
			instructedTool("atlas_open", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasOpenSentence},
				v1.ToolServer_TOOL_SERVER_FULL, v1.ToolServer_TOOL_SERVER_READING),
			instructedTool("atlas_fold", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasFoldSentence}, v1.ToolServer_TOOL_SERVER_FULL),
			readOnlyTool("atlas_glance", glanceSentence),
		)),
		mcpCaller(),
	)
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})

	specs := agenttoolsbridge.NewCatalog(host, agenttoolsbridge.Server(v1.ToolServer_TOOL_SERVER_READING)).ListTools(t.Context(), mcpPlugin)

	if len(specs) != 1 || specs[0].GetName() != "atlas_open" || !slices.Equal(specs[0].GetInstructions(), []string{atlasOpenSentence}) {
		t.Errorf("the reading catalog lists %v, want atlas_open alone with its sentence", specs)
	}
}
