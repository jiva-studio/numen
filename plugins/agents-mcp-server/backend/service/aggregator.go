// Package service exposes the host tools through an MCP server.
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

// ToolClient calls the host or plugin tool service.
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
		mu:         sync.RWMutex{},
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
			Meta:         nil,
			Name:         name,
			Title:        wireTool.GetTitle(),
			Description:  wireTool.GetDescription(),
			InputSchema:  sanitizeInputSchema(wireTool.GetInputSchema()),
			OutputSchema: nil,
			Icons:        nil,
			Annotations: &sdk.ToolAnnotations{
				DestructiveHint: nil,
				IdempotentHint:  false,
				OpenWorldHint:   nil,
				ReadOnlyHint:    wireTool.GetEffect() == tools.ReadOnly,
				Title:           "",
			},
		}
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_DECREMENT,INTEGER_INCREMENT reason="empty schema is omitted and valid JSON schema is longer than one byte"
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
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_DECREMENT reason="RemoveTools with an empty slice is a no-op"
	if len(toRemove) > 0 {
		a.mcpServer.RemoveTools(toRemove...)
	}

	a.registered = current
	return nil
}

// describeToolError is the message for the error code the host returned for the tool.
func describeToolError(code tools.ErrorCode, name string) string {
	switch code {
	case tools.ErrorCodeUnknownTool:
		return "unknown tool: " + name
	case tools.ErrorCodeBadArguments:
		return "invalid tool arguments"
	case tools.ErrorCodeUnspecified:
		return fmt.Sprintf("tool error: %v", code)
	default:
		return fmt.Sprintf("tool error: %v", code)
	}
}

// ExecuteTool forwards a tool call to the host and translates the response. The host runs it in the
// vault it has open, without request context: the call does not originate from a Numen request.
func (a *ToolAggregator) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (*sdk.CallToolResult, error) {
	if a.client == nil {
		return newErrorResult("no host client configured"), nil
	}

	res, err := a.client.InvokeTool(ctx, tools.Call{
		Name:      name,
		Arguments: []byte(args),
		Context:   nil,
		VaultID:   "",
	})
	if err != nil {
		return nil, fmt.Errorf("call host tool %s: %w", name, err)
	}

	if res.GetError() != tools.ErrorCodeUnspecified {
		return newErrorResult(describeToolError(res.GetError(), name)), nil
	}

	wireResult := res.GetResult()
	result := &sdk.CallToolResult{
		Meta:              nil,
		Content:           nil,
		StructuredContent: nil,
		IsError:           wireResult.GetIsRefusal(),
		InputRequests:     nil,
		RequestState:      "",
	}
	if len(wireResult.GetStructured()) > 0 {
		result.StructuredContent = json.RawMessage(wireResult.GetStructured())
	}

	text := wireResult.GetText()
	if text == "" {
		text = string(wireResult.GetStructured())
	}
	if text != "" {
		result.Content = []sdk.Content{newTextContent(text)}
	}

	return result, nil
}

// newTextContent creates a text content block.
func newTextContent(text string) *sdk.TextContent {
	return &sdk.TextContent{Text: text, Meta: nil, Annotations: nil}
}

// newErrorResult creates a tool result that reports message as an error.
func newErrorResult(message string) *sdk.CallToolResult {
	return &sdk.CallToolResult{
		Meta:              nil,
		Content:           []sdk.Content{newTextContent(message)},
		StructuredContent: nil,
		IsError:           true,
		InputRequests:     nil,
		RequestState:      "",
	}
}

// sanitizeInputSchema ensures the schema is non-nil and specifies an object type.
func sanitizeInputSchema(raw []byte) json.RawMessage {
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return json.RawMessage(`{"type":"object"}`)
	}
	if t, ok := schema["type"].(string); !ok || !strings.EqualFold(t, "object") {
		schema["type"] = "object"
		// gomutants:disable-next-line BRANCH_IF reason="json.Marshal on an unmarshaled map with a string entry cannot fail"
		if encoded, err := json.Marshal(schema); err == nil {
			return json.RawMessage(encoded)
		}
	}
	return json.RawMessage(raw)
}
