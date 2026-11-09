package container_test

import (
	"net/http"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

const atlasToolsPlugin = "md.test.atlastools"

func newToolRequiring(name string, effect v1.ToolEffect, requires ...string) *v1.PluginTool {
	tool := newPluginTool(name, effect)
	tool.Requires = requires
	return tool
}

func getRequireCases() map[string]struct {
	tool   *v1.PluginTool
	effect []string
} {
	return map[string]struct {
		tool   *v1.PluginTool
		effect []string
	}{
		"vault:manage": {
			tool:   newToolRequiring("atlas_write", v1.ToolEffect_TOOL_EFFECT_READ_WRITE, "vault:manage"),
			effect: []string{"vault:read", "vault:write"},
		},
		"index:read": {
			tool:   newToolRequiring("atlas_index", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, "index:read"),
			effect: []string{"vault:read"},
		},
		"window:focus": {
			tool:   newToolRequiring("atlas_spotlight", v1.ToolEffect_TOOL_EFFECT_READ_ONLY, "window:focus"),
			effect: []string{"vault:read"},
		},
	}
}

func TestCallerWithoutRequiredPermissionCannotListOrCallTool(t *testing.T) {
	for permission, c := range getRequireCases() {
		t.Run(permission, func(t *testing.T) {
			served := &toolPlugin{tools: []*v1.PluginTool{newPluginTool("atlas_open", v1.ToolEffect_TOOL_EFFECT_READ_ONLY), c.tool}}
			host := hostOver(t,
				toolsAt(atlasToolsPlugin, []string{"vault:read", "vault:write", permission}, served),
				testPlugin{id: mcpPlugin, permissions: []string{"vault:read", "vault:write"}, mount: func(*http.ServeMux) {}},
			)
			client := openToolsOfHost(t, host)

			if names := getToolNames(listToolsOfHost(t, client)); !slices.Equal(names, []string{"atlas_open"}) {
				t.Errorf("a caller without %s is listed the tools %v", permission, names)
			}
			res, err := client.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{Name: c.tool.GetName()}))
			if err != nil {
				t.Fatal(err)
			}
			if res.Msg.GetError() != v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL {
				t.Errorf("a call of %s without %s returned %v, want UNKNOWN_TOOL", c.tool.GetName(), permission, res.Msg)
			}
			if requests := served.getRequests(); len(requests) != 0 {
				t.Errorf("the plugin was called with %v", requests)
			}
		})
	}
}

func TestACallerWithTheRequiredPermissionListsAndCallsTheTool(t *testing.T) {
	for permission, c := range getRequireCases() {
		t.Run(permission, func(t *testing.T) {
			served := &toolPlugin{tools: []*v1.PluginTool{newPluginTool("atlas_open", v1.ToolEffect_TOOL_EFFECT_READ_ONLY), c.tool}}
			host := hostOver(t,
				toolsAt(atlasToolsPlugin, []string{"vault:read", "vault:write", permission}, served),
				testPlugin{id: mcpPlugin, permissions: append(slices.Clone(c.effect), permission), mount: func(*http.ServeMux) {}},
			)
			client := openToolsOfHost(t, host)

			if names := getToolNames(listToolsOfHost(t, client)); !slices.Equal(names, []string{"atlas_open", c.tool.GetName()}) {
				t.Errorf("a caller with %s is listed the tools %v", permission, names)
			}
			res, err := client.CallTool(t.Context(), connect.NewRequest(&v1.CallToolRequest{Name: c.tool.GetName()}))
			if err != nil {
				t.Fatal(err)
			}
			if res.Msg.GetError() != v1.ToolErrorCode_TOOL_ERROR_CODE_UNSPECIFIED || res.Msg.GetResult() == nil {
				t.Errorf("a call of %s with %s returned %v", c.tool.GetName(), permission, res.Msg)
			}
			if requests := served.getRequests(); len(requests) != 1 || requests[0].GetName() != c.tool.GetName() {
				t.Errorf("the plugin was called with %v", requests)
			}
		})
	}
}
