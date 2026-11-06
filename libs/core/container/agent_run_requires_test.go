package container_test

import (
	"slices"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/container"
	"github.com/jiva-studio/numen/libs/core/port/agentport"
)

// agentWith is the agent plugin of the test host holding the permissions given.
func agentWith(agent *toolAgent, permissions ...string) testPlugin {
	plugin := agentAt(agent)
	plugin.permissions = permissions
	return plugin
}

// runReshelve runs the agent, which calls atlas_reshelve, on the editor over a tools plugin offering
// atlas_open and atlas_reshelve, the second requiring vault:manage, and returns the tool plugin.
func runReshelve(t *testing.T, agent *toolAgent, agentPermissions ...string) *toolPlugin {
	t.Helper()
	served := &toolPlugin{tools: []*v1.PluginTool{
		newPluginTool("atlas_open", v1.ToolEffect_TOOL_EFFECT_READ_ONLY),
		requiringTool("atlas_reshelve", v1.ToolEffect_TOOL_EFFECT_READ_WRITE, "vault:manage"),
	}}
	host := hostOver(t,
		agentWith(agent, agentPermissions...),
		toolsAt(atlasToolsPlugin, []string{"vault:read", "vault:write", "vault:manage"}, served),
	)
	assembly := container.NewEditorAssembly(container.NewConfig(container.Config{PluginHost: host}), nil, nil)

	runToEnd(t, assembly.GetAgent(), agentport.Task{Question: "Shelve the atlas of rivers.", Agent: "alpha"})
	return served
}

func TestAnAgentRunWithoutVaultManageIsNotServedAToolRequiringIt(t *testing.T) {
	agent := &toolAgent{calls: []toolCall{{name: "atlas_reshelve", arguments: `{}`}}}

	served := runReshelve(t, agent, "vault:read", "vault:write")

	if names := agent.getToolNames(); slices.Contains(names, "atlas_reshelve") {
		t.Errorf("an agent plugin without vault:manage is served %v", names)
	}
	if requests := served.getRequests(); len(requests) != 0 {
		t.Errorf("the tool plugin ran %v for an agent plugin without vault:manage", requests)
	}
}

func TestAnAgentRunWithVaultManageIsServedAToolRequiringItAndTheCallReachesThePlugin(t *testing.T) {
	agent := &toolAgent{calls: []toolCall{{name: "atlas_reshelve", arguments: `{}`}}}

	served := runReshelve(t, agent, "vault:read", "vault:write", "vault:manage")

	if names := agent.getToolNames(); !slices.Contains(names, "atlas_reshelve") {
		t.Errorf("an agent plugin holding vault:manage is served %v, want atlas_reshelve among them", names)
	}
	if requests := served.getRequests(); len(requests) != 1 || requests[0].GetName() != "atlas_reshelve" {
		t.Errorf("the tool plugin ran %v, want one call of atlas_reshelve", requests)
	}
}
