package container_test

import (
	"slices"
	"testing"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
)

func newAgentPlugin(agent *toolAgent, permissions ...string) testPlugin {
	plugin := agentAt(agent)
	plugin.permissions = permissions
	return plugin
}

func runWrite(t *testing.T, agent *toolAgent, agentPermissions ...string) *toolPlugin {
	t.Helper()
	served := &toolPlugin{tools: []*v1.PluginTool{
		newPluginTool("atlas_open", v1.ToolEffect_TOOL_EFFECT_READ_ONLY),
		newToolRequiring("atlas_write", v1.ToolEffect_TOOL_EFFECT_READ_WRITE, "vault:manage"),
	}}
	host := hostOver(t,
		newAgentPlugin(agent, agentPermissions...),
		toolsAt(atlasToolsPlugin, []string{"vault:read", "vault:write", "vault:manage"}, served),
	)
	assembly := newEditorServices(container.NewConfig(container.Config{PluginHost: container.AdaptPluginHost(host)}))

	runToEnd(t, assembly.GetAgent(), agentdomain.Task{Question: "Shelve the atlas of rivers.", Agent: "alpha"})
	return served
}

func TestAnAgentRunWithoutVaultManageIsNotServedAToolRequiringIt(t *testing.T) {
	agent := &toolAgent{calls: []toolCall{{name: "atlas_write", arguments: `{}`}}}

	served := runWrite(t, agent, "vault:read", "vault:write")

	if names := agent.getToolNames(); slices.Contains(names, "atlas_write") {
		t.Errorf("an agent plugin without vault:manage is served %v", names)
	}
	if requests := served.getRequests(); len(requests) != 0 {
		t.Errorf("the tool plugin ran %v for an agent plugin without vault:manage", requests)
	}
}

func TestAgentRunWithVaultManageGetsToolAndCallsThePlugin(t *testing.T) {
	agent := &toolAgent{calls: []toolCall{{name: "atlas_write", arguments: `{}`}}}

	served := runWrite(t, agent, "vault:read", "vault:write", "vault:manage")

	if names := agent.getToolNames(); !slices.Contains(names, "atlas_write") {
		t.Errorf("an agent plugin with vault:manage is served %v, want atlas_write among them", names)
	}
	if requests := served.getRequests(); len(requests) != 1 || requests[0].GetName() != "atlas_write" {
		t.Errorf("the tool plugin ran %v, want one call of atlas_write", requests)
	}
}
