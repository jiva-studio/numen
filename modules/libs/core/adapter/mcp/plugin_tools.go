package mcp

import (
	"context"
	"encoding/json"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/plugintool"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// pluginToolsTimeout is how long the plugins have to list their tools when a server is built.
const pluginToolsTimeout = 10 * time.Second

// stepMeta is the key of a tool's metadata that says how a call of it is shown to a person.
const stepMeta = "numen/step"

// addPluginTools adds the tools the plugins offer to the server of the kind given. It is called
// before the tools of the core are added, so a name the core serves is the core's.
func addPluginTools(ctx context.Context, server *sdk.Server, core Core, which plugintool.Server) {
	ctx, cancel := context.WithTimeout(ctx, pluginToolsTimeout)
	defer cancel()
	for _, tool := range core.Plugins.List.Execute(ctx) {
		if !tool.IsOn(which) || (which == plugintool.ServerReading && tool.Effect != plugintool.EffectReads) {
			continue
		}
		addPluginTool(server, core, tool)
	}
}

func addPluginTool(server *sdk.Server, core Core, tool plugintool.Tool) {
	call := core.Plugins.Call
	canonMap := buildSchemaCanonicalMap(tool.InputSchema)
	declared := &sdk.Tool{
		Name:        tool.Name,
		Title:       tool.Title,
		Description: tool.Description,
		InputSchema: json.RawMessage(tool.InputSchema),
		Annotations: &sdk.ToolAnnotations{ReadOnlyHint: tool.Effect == plugintool.EffectReads},
		Meta:        sdk.Meta{stepMeta: int(getToolStepKind(tool))},
	}
	if len(tool.OutputSchema) > 0 {
		declared.OutputSchema = json.RawMessage(tool.OutputSchema)
	}
	server.AddTool(declared, func(ctx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		var arguments json.RawMessage
		if req.Params != nil {
			arguments = req.Params.Arguments
		}
		normalized, err := normalizeArguments(arguments, canonMap)
		if err != nil {
			return newErrorResult(err), nil
		}
		answer, err := call.Execute(ctx, tool.PluginID, tool.Name, normalized, getAsked(core))
		if err != nil {
			return newErrorResult(err), nil
		}
		return readPluginToolResult(answer), nil
	})
}

// readPluginToolResult is what a tool answered, as a result an agent reads.
func readPluginToolResult(answer plugintool.Result) *sdk.CallToolResult {
	result := &sdk.CallToolResult{IsError: answer.IsRefusal}
	if len(answer.Structured) > 0 {
		result.StructuredContent = json.RawMessage(answer.Structured)
	}
	text := answer.Text
	if text == "" && len(answer.Structured) > 0 {
		text = string(answer.Structured)
	}
	if text != "" {
		result.Content = []sdk.Content{&sdk.TextContent{Text: text}}
	}
	return result
}

// getToolStepKind is how a call of the tool is shown to a person.
func getToolStepKind(tool plugintool.Tool) domain.StepKind {
	switch tool.Step {
	case plugintool.StepSearch:
		return domain.StepSearch
	case plugintool.StepRemove:
		return domain.StepRemove
	case plugintool.StepMove:
		return domain.StepMove
	case plugintool.StepByEffect:
	}
	if tool.Effect == plugintool.EffectWrites {
		return domain.StepEdit
	}
	return domain.StepRead
}

// buildSchemaCanonicalMap maps the canonical form of each property a schema names to the property.
func buildSchemaCanonicalMap(schema []byte) map[string]string {
	var parsed struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(schema, &parsed); err != nil {
		return nil
	}
	canon := make(map[string]string, len(parsed.Properties))
	for name := range parsed.Properties {
		canon[canonicalKey(name)] = name
	}
	return canon
}

// askedVault is the key of the vault an agent works, among what the plugins are told of the window
// the agent was started from.
const askedVault = "vault"

// getAsked is what the plugins are told of the window the agent was started from: the vault the
// agent works, and what the person is looking at there.
func getAsked(core Core) map[string]string {
	asked := map[string]string{}
	if core.Showing != nil {
		if showing := core.Showing().Vault; showing.ID != "" {
			asked[askedVault] = string(showing.ID)
		}
	}
	if core.GetAsked != nil {
		for key, value := range core.GetAsked() {
			asked[key] = value
		}
	}
	return asked
}
