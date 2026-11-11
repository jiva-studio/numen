package container_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	vaultapi "github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/pluginhost"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/agenttoolsbridge"
	"github.com/jiva-studio/numen/libs/note/domain"
)

const (
	atlasPlugin  = "md.test.atlas"
	zitherPlugin = "md.test.zither"

	atlasOpenSentence   = "Open the atlas with atlas_open before you name a country."
	atlasFindSentence   = "Find a map with atlas_find by its title."
	atlasSharedSentence = "Close the atlas with atlas_close or atlas_find when the reading ends."
	zitherTuneSentence  = "Tune the zither with zither_tune, one string at a time."
)

// toolPluginFixture is a plugin that lists the tools it is given, or fails to list when it is given
// an error.
type toolPluginFixture struct {
	numenv1connect.UnimplementedPluginToolServiceHandler
	tools []*v1.PluginTool
	err   error
}

func (p *toolPluginFixture) ListTools(
	context.Context, *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	if p.err != nil {
		return nil, connect.NewError(connect.CodeInternal, p.err)
	}
	return connect.NewResponse(&v1.ListToolsResponse{Tools: p.tools}), nil
}

func getInstructionsAt(id string, permissions []string, served *toolPluginFixture) testPlugin {
	return testPlugin{
		id: id, capabilities: []string{"tools"}, permissions: permissions,
		mount: func(mux *http.ServeMux) { mux.Handle(numenv1connect.NewPluginToolServiceHandler(served)) },
	}
}

func newToolWithInstructions(name string, effect v1.ToolEffect, sentences []string, servers ...v1.ToolServer) *v1.PluginTool {
	tool := newPluginTool(name, effect, servers...)
	tool.Instructions = sentences
	return tool
}

func newToolsPlugin(tools ...*v1.PluginTool) *toolPluginFixture {
	return &toolPluginFixture{tools: tools}
}

func newReadOnlyTool(name string, sentences ...string) *v1.PluginTool {
	return newToolWithInstructions(name, v1.ToolEffect_TOOL_EFFECT_READ_ONLY, sentences)
}

func newMCPCaller() testPlugin {
	return testPlugin{id: mcpPlugin, permissions: []string{"vault:read"}, mount: func(*http.ServeMux) {}}
}

func openToolsOfHost(t *testing.T, host *pluginhost.Host) numenv1connect.PluginToolServiceClient {
	t.Helper()
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})
	container.AdaptPluginHost((*container.PluginRoles)(host)).Endpoint.SetPluginVault(&vaultapi.Access{Switch: shownVault{vault: domain.Vault{ID: "v-orchard", Name: "Orchard"}}})
	port, err := host.Services.OpenHostEndpoint(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	caller, err := host.Activation.EnsureActive(t.Context(), mcpPlugin)
	if err != nil {
		t.Fatal(err)
	}
	return numenv1connect.NewPluginToolServiceClient(
		&http.Client{Transport: bearerTransport{token: caller.Token}}, fmt.Sprintf("http://127.0.0.1:%d", port))
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
		getInstructionsAt(zitherPlugin, []string{"vault:read"}, newToolsPlugin(newReadOnlyTool("zither_tune", zitherTuneSentence))),
		getInstructionsAt(atlasPlugin, []string{"vault:read"}, newToolsPlugin(
			newReadOnlyTool("atlas_open", atlasOpenSentence),
			newReadOnlyTool("atlas_find", atlasFindSentence, atlasSharedSentence),
			newReadOnlyTool("atlas_close", atlasSharedSentence),
		)),
		newMCPCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if names := getToolNames(listing); !slices.Equal(names, []string{"atlas_open", "atlas_find", "atlas_close", "zither_tune"}) {
		t.Fatalf("the host lists the tools %v", names)
	}
	want := map[string][]string{
		"atlas_open":  {atlasOpenSentence},
		"atlas_find":  {atlasFindSentence, atlasSharedSentence},
		"atlas_close": {atlasSharedSentence},
		"zither_tune": {zitherTuneSentence},
	}
	for name, sentences := range getSentencesByTool(listing) {
		if !slices.Equal(sentences, want[name]) {
			t.Errorf("the host lists %s with the sentences %q, want %q", name, sentences, want[name])
		}
	}
}

// The MCP plugin has vault:read only, so a read-write tool is not listed to it, and the host
// endpoint offers the full server, so a tool offered on the reading server only is not listed.
func TestListedToolsHaveNoSentenceOfAnOmittedTool(t *testing.T) {
	burnSentence := "Burn the old maps with atlas_burn after you copy them."
	lensSentence := "Read the small print with atlas_lens."
	host := hostOver(t,
		getInstructionsAt(atlasPlugin, []string{"vault:read", "vault:write"}, newToolsPlugin(
			newReadOnlyTool("atlas_open", atlasOpenSentence),
			newToolWithInstructions("atlas_burn", v1.ToolEffect_TOOL_EFFECT_READ_WRITE, []string{burnSentence}),
			newToolWithInstructions("atlas_lens", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{lensSentence}, v1.ToolServer_TOOL_SERVER_READING),
		)),
		newMCPCaller(),
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
	failingPlugin := newToolsPlugin(newReadOnlyTool("bellows_pump", "Pump the bellows with bellows_pump."))
	failingPlugin.err = errors.New("the bellows are torn")
	host := hostOver(t,
		getInstructionsAt(atlasPlugin, []string{"vault:read"}, newToolsPlugin(newReadOnlyTool("atlas_open", atlasOpenSentence))),
		getInstructionsAt("md.test.bellows", []string{"vault:read"}, failingPlugin),
		newMCPCaller(),
	)

	listing := listToolsOfHost(t, openToolsOfHost(t, host))

	if names := getToolNames(listing); !slices.Equal(names, []string{"atlas_open"}) {
		t.Errorf("the host lists the tools %v", names)
	}
}

func TestDisablingAPluginRemovesItsToolsAndTheirSentences(t *testing.T) {
	host := hostOver(t,
		getInstructionsAt(atlasPlugin, []string{"vault:read"}, newToolsPlugin(newReadOnlyTool("atlas_open", atlasOpenSentence))),
		getInstructionsAt(zitherPlugin, []string{"vault:read"}, newToolsPlugin(newReadOnlyTool("zither_tune", zitherTuneSentence))),
		newMCPCaller(),
	)
	client := openToolsOfHost(t, host)
	if before := getToolNames(listToolsOfHost(t, client)); !slices.Equal(before, []string{"atlas_open", "zither_tune"}) {
		t.Fatalf("before the plugin is disabled the host lists the tools %v", before)
	}

	if err := setEnabled(t.Context(), host, zitherPlugin, false); err != nil {
		t.Fatal(err)
	}

	if names := getToolNames(listToolsOfHost(t, client)); !slices.Equal(names, []string{"atlas_open"}) {
		t.Errorf("after the plugin is disabled the host lists the tools %v", names)
	}
}

func TestTheToolsOfTheCallerItselfAreNotListed(t *testing.T) {
	caller := getInstructionsAt(mcpPlugin, []string{"vault:read"}, newToolsPlugin(newReadOnlyTool("mirror_look", "Look in the mirror with mirror_look.")))
	host := hostOver(t,
		getInstructionsAt(atlasPlugin, []string{"vault:read"}, newToolsPlugin(newReadOnlyTool("atlas_open", atlasOpenSentence))),
		caller,
	)

	if names := getToolNames(listToolsOfHost(t, openToolsOfHost(t, host))); !slices.Equal(names, []string{"atlas_open"}) {
		t.Errorf("the host lists the caller the tools %v", names)
	}
}

func TestReadingCatalogListsOnlyReadingToolsWithSentences(t *testing.T) {
	glanceSentence := "Glance at the atlas index with atlas_glance."
	host := hostOver(t,
		getInstructionsAt(atlasPlugin, []string{"vault:read"}, newToolsPlugin(
			newToolWithInstructions("atlas_open", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasOpenSentence},
				v1.ToolServer_TOOL_SERVER_FULL, v1.ToolServer_TOOL_SERVER_READING),
			newToolWithInstructions("atlas_find", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{atlasFindSentence}, v1.ToolServer_TOOL_SERVER_FULL),
			newReadOnlyTool("atlas_glance", glanceSentence),
		)),
		newMCPCaller(),
	)
	t.Cleanup(func() {
		if err := host.Teardown.TeardownAll(context.WithoutCancel(t.Context()), time.Millisecond); err != nil {
			t.Error(err)
		}
	})

	specs := agenttoolsbridge.NewCatalog(pluginhost.Join(host), agenttoolsbridge.Server(v1.ToolServer_TOOL_SERVER_READING)).ListTools(t.Context(), mcpPlugin)

	if len(specs) != 1 || specs[0].GetName() != "atlas_open" || !slices.Equal(specs[0].GetInstructions(), []string{atlasOpenSentence}) {
		t.Errorf("the reading catalog lists %v, want atlas_open alone with its sentence", specs)
	}
}

func runInApp(t *testing.T, instructions string, plugins ...testPlugin) string {
	t.Helper()
	agent := &toolAgent{}
	host := hostOver(t, append([]testPlugin{agentAt(agent)}, plugins...)...)
	assembly := newEditorServices(container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost((*container.PluginRoles)(host))}))

	runToEnd(t, assembly.GetAgent(), agentdomain.Task{Question: "q", Agent: "alpha", Instructions: instructions})

	requests := agent.getRequests()
	if len(requests) != 1 {
		t.Fatalf("the agent received %d requests, want one", len(requests))
	}
	return requests[0].GetInstructions()
}

func TestInAppRunReceivesDistinctToolSentencesAfterInstructions(t *testing.T) {
	got := runInApp(t, "Answer the gardener in short sentences.",
		getInstructionsAt(zitherPlugin, []string{"vault:read"}, newToolsPlugin(newReadOnlyTool("zither_tune", zitherTuneSentence))),
		getInstructionsAt(atlasPlugin, []string{"vault:read"}, newToolsPlugin(
			newReadOnlyTool("atlas_open", atlasOpenSentence),
			newReadOnlyTool("atlas_find", atlasFindSentence, atlasSharedSentence),
			newReadOnlyTool("atlas_close", atlasSharedSentence),
		)),
	)

	want := "Answer the gardener in short sentences.\n\n" +
		strings.Join([]string{atlasOpenSentence, atlasFindSentence, atlasSharedSentence, zitherTuneSentence}, " ")
	if got != want {
		t.Errorf("the run received the instructions %q, want %q", got, want)
	}
}

func TestInAppRunWithoutInstructionsReceivesOnlyToolSentences(t *testing.T) {
	got := runInApp(t, "",
		getInstructionsAt(atlasPlugin, []string{"vault:read"}, newToolsPlugin(
			newReadOnlyTool("atlas_open", atlasOpenSentence),
			newReadOnlyTool("atlas_find", atlasFindSentence),
		)),
	)

	if want := atlasOpenSentence + " " + atlasFindSentence; got != want {
		t.Errorf("the run received the instructions %q, want %q", got, want)
	}
}

// The atlas plugin has vault:read only, so its read-write tool is not served, and a tool offered on
// the reading server only is not served on the full server an in-app run uses.
func TestAToolLeftOutOfAnInAppRunAddsNoSentenceToItsInstructions(t *testing.T) {
	burnSentence := "Burn the old maps with atlas_burn after you copy them."
	lensSentence := "Read the small print with atlas_lens."
	got := runInApp(t, "Answer the gardener in short sentences.",
		getInstructionsAt(atlasPlugin, []string{"vault:read"}, newToolsPlugin(
			newReadOnlyTool("atlas_open", atlasOpenSentence),
			newToolWithInstructions("atlas_burn", v1.ToolEffect_TOOL_EFFECT_READ_WRITE, []string{burnSentence}),
			newToolWithInstructions("atlas_lens", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, []string{lensSentence}, v1.ToolServer_TOOL_SERVER_READING),
		)),
	)

	if want := "Answer the gardener in short sentences.\n\n" + atlasOpenSentence; got != want {
		t.Errorf("the run received the instructions %q, want %q", got, want)
	}
}

func TestInAppRunWithoutToolSentencesKeepsItsInstructions(t *testing.T) {
	got := runInApp(t, "Answer the gardener in short sentences.",
		getInstructionsAt(atlasPlugin, []string{"vault:read"}, newToolsPlugin(newReadOnlyTool("atlas_open"))),
	)

	if want := "Answer the gardener in short sentences."; got != want {
		t.Errorf("the run received the instructions %q, want %q", got, want)
	}
}

func TestAnInAppRunWithNoToolsKeepsItsInstructions(t *testing.T) {
	if got, want := runInApp(t, "Answer the gardener in short sentences."), "Answer the gardener in short sentences."; got != want {
		t.Errorf("the run received the instructions %q, want %q", got, want)
	}
}
