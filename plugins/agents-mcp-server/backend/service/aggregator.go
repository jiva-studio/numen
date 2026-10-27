package service

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// ToolClient reaches the host or plugin tool service.
type ToolClient interface {
	ListToolInfos(ctx context.Context) ([]*tools.ToolInfo, error)
	InvokeTool(ctx context.Context, call tools.Call) (*tools.CallResponse, error)
}

// ToolAggregator discovers tools from the host and exposes them through an MCP server.
type ToolAggregator struct {
	client     ToolClient
	mcpServer  *sdk.Server
	mu         sync.RWMutex
	registered map[string]bool
}

// NewToolAggregator creates a tool aggregator for the given host client and MCP server.
func NewToolAggregator(client ToolClient, mcpServer *sdk.Server) *ToolAggregator {
	return &ToolAggregator{
		client:     client,
		mcpServer:  mcpServer,
		registered: make(map[string]bool),
	}
}

// GetRegisteredTools returns the sorted list of currently registered tool names.
func (a *ToolAggregator) GetRegisteredTools() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	names := make([]string, 0, len(a.registered))
	for name := range a.registered {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// SyncTools queries the host for available tools and updates the MCP server registry.
func (a *ToolAggregator) SyncTools(ctx context.Context) error {
	if a.client == nil || a.mcpServer == nil {
		return nil
	}

	wireTools, err := a.client.ListToolInfos(ctx)
	if err != nil {
		return fmt.Errorf("list host tools: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	current := make(map[string]bool)
	for _, wireTool := range wireTools {
		name := wireTool.GetName()
		if name == "" {
			continue
		}
		current[name] = true

		declared := &sdk.Tool{
			Name:        name,
			Title:       wireTool.GetTitle(),
			Description: wireTool.GetDescription(),
			InputSchema: sanitizeInputSchema(wireTool.GetInputSchema()),
			Annotations: &sdk.ToolAnnotations{
				ReadOnlyHint: wireTool.GetEffect() == tools.ReadOnly,
			},
		}
		if len(wireTool.GetOutputSchema()) > 0 {
			declared.OutputSchema = json.RawMessage(wireTool.GetOutputSchema())
		}

		toolName := name
		a.mcpServer.AddTool(declared, func(callCtx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			var args json.RawMessage
			if req.Params != nil {
				args = req.Params.Arguments
			}
			return a.ExecuteTool(callCtx, toolName, args)
		})
	}

	var toRemove []string
	for oldName := range a.registered {
		if !current[oldName] {
			toRemove = append(toRemove, oldName)
		}
	}
	if len(toRemove) > 0 {
		a.mcpServer.RemoveTools(toRemove...)
	}

	a.registered = current
	return nil
}

// ExecuteTool forwards a tool call to the host and translates the response. The host runs it in the
// vault it shows, with no context: no question was asked in Numen.
func (a *ToolAggregator) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (*sdk.CallToolResult, error) {
	if a.client == nil {
		return &sdk.CallToolResult{
			IsError: true,
			Content: []sdk.Content{&sdk.TextContent{Text: "no host client configured"}},
		}, nil
	}

	res, err := a.client.InvokeTool(ctx, tools.Call{
		Name:      name,
		Arguments: []byte(args),
	})
	if err != nil {
		return nil, fmt.Errorf("call host tool %s: %w", name, err)
	}

	if res.GetError() != tools.ErrorCodeUnspecified {
		var errMsg string
		switch res.GetError() {
		case tools.ErrorCodeUnknownTool:
			errMsg = "unknown tool: " + name
		case tools.ErrorCodeBadArguments:
			errMsg = "invalid tool arguments"
		default:
			errMsg = fmt.Sprintf("tool error: %v", res.GetError())
		}
		return &sdk.CallToolResult{
			IsError: true,
			Content: []sdk.Content{&sdk.TextContent{Text: errMsg}},
		}, nil
	}

	wireResult := res.GetResult()
	if wireResult == nil {
		return &sdk.CallToolResult{}, nil
	}

	result := &sdk.CallToolResult{IsError: wireResult.GetIsRefusal()}
	if len(wireResult.GetStructured()) > 0 {
		result.StructuredContent = json.RawMessage(wireResult.GetStructured())
	}

	text := wireResult.GetText()
	if text == "" && len(wireResult.GetStructured()) > 0 {
		text = string(wireResult.GetStructured())
	}
	if text != "" {
		result.Content = []sdk.Content{&sdk.TextContent{Text: text}}
	}

	return result, nil
}

// sanitizeInputSchema ensures the schema is non-nil and specifies an object type.
func sanitizeInputSchema(raw []byte) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{"type":"object"}`)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return json.RawMessage(`{"type":"object"}`)
	}
	if t, ok := schema["type"].(string); !ok || !strings.EqualFold(t, "object") {
		schema["type"] = "object"
		encoded, err := json.Marshal(schema)
		if err == nil {
			return json.RawMessage(encoded)
		}
		return json.RawMessage(`{"type":"object"}`)
	}
	return json.RawMessage(raw)
}
