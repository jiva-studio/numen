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

	atlasInstructions  = "Open the atlas with atlas_open before you name a country."
	zitherInstructions = "Tune the zither with zither_tune, then play one string at a time."
)

// instructedPlugin is a plugin that lists its tools with the instructions it is given, or fails to
// list when it is given an error.
type instructedPlugin struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
	tools        []*v1.PluginTool
	instructions string
	err          error
}

func (p *instructedPlugin) ListTools(
	context.Context, *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	if p.err != nil {
		return nil, connect.NewError(connect.CodeInternal, p.err)
	}
	return connect.NewResponse(&v1.ListToolsResponse{Tools: p.tools, Instructions: p.instructions}), nil
}

func instructedAt(id string, permissions []string, served *instructedPlugin) testPlugin {
	return testPlugin{
		id: id, capabilities: []string{"tools"}, permissions: permissions,
		mount: func(mux *http.ServeMux) { mux.Handle(numenv1connect.NewPluginToolServiceHandler(served)) },
	}
}

func readOnlyPlugin(name, instructions string) *instructedPlugin {
	return &instructedPlugin{
		tools:        []*v1.PluginTool{newPluginTool(name, v1.ToolEffect_TOOL_EFFECT_READ_ONLY)},
		instructions: instructions,
	}
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

func TestTheHostJoinsInstructionsInPluginIdOrder(t *testing.T) {
	host := hostOver(t,
		instructedAt(zitherPlugin, []string{"vault:read"}, readOnlyPlugin("zither_tune", zitherInstructions)),
		instructedAt(atlasPlugin, []string{"vault:read"}, readOnlyPlugin("atlas_open", atlasInstructions)),
		mcpCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if want := atlasInstructions + "\n\n" + zitherInstructions; listing.GetInstructions() != want {
		t.Errorf("the host lists the instructions %q, want %q", listing.GetInstructions(), want)
	}
	if names := getToolNames(listing); !slices.Equal(names, []string{"atlas_open", "zither_tune"}) {
		t.Errorf("the host lists the tools %v", names)
	}
}

func TestAPluginThatFailsToListAddsNoInstructions(t *testing.T) {
	broken := readOnlyPlugin("bellows_pump", "Pump the bellows with bellows_pump.")
	broken.err = errors.New("the bellows are torn")
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read"}, readOnlyPlugin("atlas_open", atlasInstructions)),
		instructedAt("md.test.bellows", []string{"vault:read"}, broken),
		mcpCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if listing.GetInstructions() != atlasInstructions {
		t.Errorf("the host lists the instructions %q, want only %q", listing.GetInstructions(), atlasInstructions)
	}
}

func TestDisablingAPluginRemovesItsInstructions(t *testing.T) {
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read"}, readOnlyPlugin("atlas_open", atlasInstructions)),
		instructedAt(zitherPlugin, []string{"vault:read"}, readOnlyPlugin("zither_tune", zitherInstructions)),
		mcpCaller(),
	)
	client := openToolsOfHost(t, host)
	if before := listToolsOfHost(t, client).GetInstructions(); before != atlasInstructions+"\n\n"+zitherInstructions {
		t.Fatalf("before the plugin is disabled the host lists the instructions %q", before)
	}

	if err := host.SetPluginEnabled(t.Context(), zitherPlugin, false); err != nil {
		t.Fatal(err)
	}
	listing := listToolsOfHost(t, client)

	if listing.GetInstructions() != atlasInstructions {
		t.Errorf("after the plugin is disabled the host lists the instructions %q, want %q", listing.GetInstructions(), atlasInstructions)
	}
	if names := getToolNames(listing); !slices.Equal(names, []string{"atlas_open"}) {
		t.Errorf("after the plugin is disabled the host lists the tools %v", names)
	}
}

// The MCP plugin holds vault:read only, so a read-write tool is not listing to it; the catalog offers
// the full server, so a tool offered on the reading server only is not listing; and of two tools of
// one name the first plugin's is kept.
func TestAPluginWhoseToolsAreAllLeftOutAddsNoInstructions(t *testing.T) {
	writer := &instructedPlugin{
		tools:        []*v1.PluginTool{newPluginTool("ledger_write", v1.ToolEffect_TOOL_EFFECT_READ_WRITE)},
		instructions: "Write the ledger with ledger_write in ink.",
	}
	reader := &instructedPlugin{
		tools: []*v1.PluginTool{
			newPluginTool("lens_view", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, v1.ToolServer_TOOL_SERVER_READING),
		},
		instructions: "Hold the lens with lens_view close to the page.",
	}
	echo := readOnlyPlugin("atlas_open", "Echo the atlas back with atlas_open.")
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read"}, readOnlyPlugin("atlas_open", atlasInstructions)),
		instructedAt("md.test.echo", []string{"vault:read"}, echo),
		instructedAt("md.test.ledger", []string{"vault:read", "vault:write"}, writer),
		instructedAt("md.test.lens", []string{"vault:read"}, reader),
		mcpCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if names := getToolNames(listing); !slices.Equal(names, []string{"atlas_open"}) {
		t.Fatalf("the host lists the tools %v", names)
	}
	if listing.GetInstructions() != atlasInstructions {
		t.Errorf("the host lists the instructions %q, want only %q", listing.GetInstructions(), atlasInstructions)
	}
}

func TestAPluginWithToolsButNoInstructionsAddsNoSeparator(t *testing.T) {
	host := hostOver(t,
		instructedAt("md.test.anvil", []string{"vault:read"}, readOnlyPlugin("anvil_strike", "")),
		instructedAt(atlasPlugin, []string{"vault:read"}, readOnlyPlugin("atlas_open", atlasInstructions)),
		instructedAt("md.test.kettle", []string{"vault:read"}, readOnlyPlugin("kettle_boil", "")),
		instructedAt(zitherPlugin, []string{"vault:read"}, readOnlyPlugin("zither_tune", zitherInstructions)),
		instructedAt("md.test.zz", []string{"vault:read"}, readOnlyPlugin("zz_sleep", "")),
		mcpCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if want := atlasInstructions + "\n\n" + zitherInstructions; listing.GetInstructions() != want {
		t.Errorf("the host lists the instructions %q, want %q", listing.GetInstructions(), want)
	}
	if names := getToolNames(listing); len(names) != 5 {
		t.Errorf("the host lists the tools %v", names)
	}
}

func TestPluginsWithoutInstructionsListNone(t *testing.T) {
	host := hostOver(t,
		instructedAt("md.test.anvil", []string{"vault:read"}, readOnlyPlugin("anvil_strike", "")),
		instructedAt("md.test.kettle", []string{"vault:read"}, readOnlyPlugin("kettle_boil", "")),
		mcpCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if listing.GetInstructions() != "" {
		t.Errorf("plugins with no instructions are listing with %q", listing.GetInstructions())
	}
}

func TestTheInstructionsOfTheCallerItselfAreNotListed(t *testing.T) {
	own := readOnlyPlugin("mirror_look", "Look in the mirror with mirror_look.")
	caller := instructedAt(mcpPlugin, []string{"vault:read"}, own)
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read"}, readOnlyPlugin("atlas_open", atlasInstructions)),
		caller,
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if listing.GetInstructions() != atlasInstructions {
		t.Errorf("the host lists the instructions %q to their own plugin, want only %q", listing.GetInstructions(), atlasInstructions)
	}
}

const (
	atlasOpenSentence   = "Open the atlas with atlas_open before you name a country."
	atlasFoldSentence   = "Fold a map with atlas_fold along its printed creases."
	atlasSharedSentence = "Close the atlas with atlas_close or atlas_fold when the reading ends."
	zitherTuneSentence  = "Tune the zither with zither_tune, one string at a time."
)

// instructedTool is a tool of a test plugin that carries the instruction sentences given.
func instructedTool(name string, effect v1.ToolEffect, sentences []string, servers ...v1.ToolServer) *v1.PluginTool {
	tool := newPluginTool(name, effect, servers...)
	tool.Instructions = sentences
	return tool
}

func toolsPlugin(tools ...*v1.PluginTool) *instructedPlugin {
	return &instructedPlugin{tools: tools}
}

// listReadingToolsOfHost lists, as the MCP plugin, the tools of the host offered on the reading server.
func listReadingToolsOfHost(t *testing.T, host *plugin.Host) ([]*v1.PluginTool, string) {
	t.Helper()
	t.Cleanup(func() {
		if err := host.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	catalog := agenttoolsbridge.NewCatalog(host, agenttoolsbridge.Server(v1.ToolServer_TOOL_SERVER_READING))
	return catalog.ListTools(t.Context(), mcpPlugin)
}

func TestTheHostJoinsTheDistinctSentencesOfAPluginsToolsIntoOneParagraph(t *testing.T) {
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read"}, toolsPlugin(
			instructedTool("atlas_open", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasOpenSentence}),
			instructedTool("atlas_fold", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasFoldSentence, atlasSharedSentence}),
			instructedTool("atlas_close", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasSharedSentence}),
		)),
		mcpCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if want := atlasOpenSentence + " " + atlasFoldSentence + " " + atlasSharedSentence; listing.GetInstructions() != want {
		t.Errorf("the host lists the instructions %q, want %q", listing.GetInstructions(), want)
	}
}

func TestTheHostListsOneParagraphOfToolSentencesPerPluginInPluginIdOrder(t *testing.T) {
	host := hostOver(t,
		instructedAt(zitherPlugin, []string{"vault:read"}, toolsPlugin(
			instructedTool("zither_tune", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{zitherTuneSentence}),
		)),
		instructedAt(atlasPlugin, []string{"vault:read"}, toolsPlugin(
			instructedTool("atlas_open", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasOpenSentence}),
			instructedTool("atlas_fold", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasFoldSentence}),
		)),
		mcpCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if want := atlasOpenSentence + " " + atlasFoldSentence + "\n\n" + zitherTuneSentence; listing.GetInstructions() != want {
		t.Errorf("the host lists the instructions %q, want %q", listing.GetInstructions(), want)
	}
}

// The MCP plugin holds vault:read only, so a read-write tool is not listed to it, and the host
// endpoint offers the full server, so a tool offered on the reading server only is not listed.
func TestTheSentenceOfAToolLeftOutIsNotListed(t *testing.T) {
	burnSentence := "Burn the old maps with atlas_burn after you copy them."
	lensSentence := "Read the small print with atlas_lens."
	ledgerSentence := "Write the ledger with ledger_write in ink."
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read", "vault:write"}, toolsPlugin(
			instructedTool("atlas_open", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasOpenSentence}),
			instructedTool("atlas_burn", v1.ToolEffect_TOOL_EFFECT_READ_WRITE, []string{burnSentence}),
			instructedTool("atlas_lens", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{lensSentence}, v1.ToolServer_TOOL_SERVER_READING),
		)),
		instructedAt("md.test.ledger", []string{"vault:read", "vault:write"}, toolsPlugin(
			instructedTool("ledger_write", v1.ToolEffect_TOOL_EFFECT_READ_WRITE, []string{ledgerSentence}),
		)),
		mcpCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if names := getToolNames(listing); !slices.Equal(names, []string{"atlas_open"}) {
		t.Fatalf("the host lists the tools %v", names)
	}
	if listing.GetInstructions() != atlasOpenSentence {
		t.Errorf("the host lists the instructions %q, want only %q", listing.GetInstructions(), atlasOpenSentence)
	}
}

func TestTheReadingCatalogListsOnlySentencesOfReadingServerTools(t *testing.T) {
	glanceSentence := "Glance at the atlas index with atlas_glance."
	host := hostOver(t,
		instructedAt(atlasPlugin, []string{"vault:read"}, toolsPlugin(
			instructedTool("atlas_open", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasOpenSentence},
				v1.ToolServer_TOOL_SERVER_FULL, v1.ToolServer_TOOL_SERVER_READING),
			instructedTool("atlas_fold", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasFoldSentence}, v1.ToolServer_TOOL_SERVER_FULL),
			instructedTool("atlas_glance", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{glanceSentence}),
		)),
		mcpCaller(),
	)

	specs, instructions := listReadingToolsOfHost(t, host)

	names := make([]string, 0, len(specs))
	for _, one := range specs {
		names = append(names, one.GetName())
	}
	if !slices.Equal(names, []string{"atlas_open"}) {
		t.Fatalf("the reading listing holds the tools %v", names)
	}
	if instructions != atlasOpenSentence {
		t.Errorf("the reading listing holds the instructions %q, want only %q", instructions, atlasOpenSentence)
	}
}
