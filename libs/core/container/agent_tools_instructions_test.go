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
