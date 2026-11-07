// Package service exposes the host tools through an MCP server.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// ToolClient calls the host or plugin tool service.
type ToolClient interface {
	ListTools(ctx context.Context) (tools.Listing, error)
	InvokeTool(ctx context.Context, call tools.Call) (*tools.CallResponse, error)
}

// ToolAggregator discovers tools from the host and exposes them through an MCP server.
type ToolAggregator struct {
	client    ToolClient
	mcpServer *sdk.Server
	mu        sync.RWMutex
	// registered maps the name of each tool on the MCP server to its declaration.
	registered map[string]*sdk.Tool
	// syncMu is locked for the whole sync, host call included, so the sync that finishes last is the
	// one that listed last.
	syncMu sync.Mutex
	// sentences are the distinct instructions of the tools registered at the latest sync, in listing
	// order.
	sentences []string
}

// NewToolAggregator creates a tool aggregator for the given host client and MCP server.
func NewToolAggregator(client ToolClient, mcpServer *sdk.Server) *ToolAggregator {
	return &ToolAggregator{
		client:     client,
		mcpServer:  mcpServer,
		mu:         sync.RWMutex{},
		syncMu:     sync.Mutex{},
		registered: make(map[string]*sdk.Tool),
		sentences:  nil,
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

// GetInstructions returns the distinct instructions of the tools registered at the latest sync, in
// listing order, joined with one space.
func (a *ToolAggregator) GetInstructions() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return strings.Join(a.sentences, " ")
}

// SyncTools queries the host for available tools and updates the MCP server registry.
func (a *ToolAggregator) SyncTools(ctx context.Context) error {
	if a.client == nil || a.mcpServer == nil {
		return nil
	}
	a.syncMu.Lock()
	defer a.syncMu.Unlock()

	listing, err := a.client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("list host tools: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	declared, sentences := readDeclarations(listing)
	a.updateTools(declared)
	a.registered = declared
	a.sentences = sentences
	return nil
}

// readDeclarations returns the MCP declarations of the listed tools by name, the last one listed
// under a name winning, and the distinct instructions of the tools in listing order.
func readDeclarations(listing tools.Listing) (map[string]*sdk.Tool, []string) {
	declared := make(map[string]*sdk.Tool)
	var sentences []string
	for _, wireTool := range listing.Tools {
		if wireTool.GetName() == "" {
			continue
		}
		declared[wireTool.GetName()] = createTool(wireTool)
		for _, sentence := range wireTool.GetInstructions() {
			if !slices.Contains(sentences, sentence) {
				sentences = append(sentences, sentence)
			}
		}
	}
	return declared, sentences
}

// updateTools adds to the MCP server the declared tools that are new or changed since the latest
// sync, and removes the registered tools that are not declared. Each change makes the MCP server
// notify the connected sessions that the tool list changed.
func (a *ToolAggregator) updateTools(declared map[string]*sdk.Tool) {
	for name, tool := range declared {
		if previous, isRegistered := a.registered[name]; isRegistered && reflect.DeepEqual(previous, tool) {
			continue
		}
		a.mcpServer.AddTool(tool, func(callCtx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			var args json.RawMessage
			if req.Params != nil {
				args = req.Params.Arguments
			}
			return a.ExecuteTool(callCtx, name, args)
		})
	}

	var toRemove []string
	for oldName := range a.registered {
		if _, isDeclared := declared[oldName]; !isDeclared {
			toRemove = append(toRemove, oldName)
		}
	}
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_DECREMENT reason="RemoveTools with an empty slice is a no-op"
	if len(toRemove) > 0 {
		a.mcpServer.RemoveTools(toRemove...)
	}
}

// createTool creates the MCP declaration of a host tool. The annotations are the hints of the tool.
func createTool(wireTool *tools.ToolInfo) *sdk.Tool {
	hints := wireTool.GetHints()
	tool := &sdk.Tool{
		Meta:         nil,
		Name:         wireTool.GetName(),
		Title:        wireTool.GetTitle(),
		Description:  wireTool.GetDescription(),
		InputSchema:  readInputSchema(wireTool.GetInputSchema()),
		OutputSchema: nil,
		Icons:        nil,
		Annotations: &sdk.ToolAnnotations{
			DestructiveHint: &hints.Destructive,
			IdempotentHint:  hints.Idempotent,
			OpenWorldHint:   &hints.OpenWorld,
			ReadOnlyHint:    hints.ReadOnly,
			Title:           "",
		},
	}
	if schema, isObject := readObject(wireTool.GetOutputSchema()); isObject && hasObjectType(schema) {
		tool.OutputSchema = json.RawMessage(wireTool.GetOutputSchema())
	}
	return tool
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
		IsError:           wireResult.GetIsError(),
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

// readInputSchema returns the input schema of a tool as a JSON object of type "object", which MCP
// requires. A JSON object of another type or none is given the type "object" and keeps its other
// keys; anything else is replaced by the empty object schema. The same rules are in
// libs/core/internal/adapter/plugin/agenttoolsbridge/endpoint.go.
func readInputSchema(raw []byte) json.RawMessage {
	schema, isObject := readObject(raw)
	if !isObject {
		return json.RawMessage(`{"type":"object"}`)
	}
	if hasObjectType(schema) {
		return json.RawMessage(raw)
	}
	schema["type"] = json.RawMessage(`"object"`)
	encoded, err := json.Marshal(schema)
	if err != nil {
		return json.RawMessage(`{"type":"object"}`)
	}
	return encoded
}

// readObject decodes raw as a JSON object, and reports false for anything else.
func readObject(raw []byte) (map[string]json.RawMessage, bool) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		return nil, false
	}
	return object, true
}

// hasObjectType reports whether the type of the schema is the string "object".
func hasObjectType(schema map[string]json.RawMessage) bool {
	var schemaType string
	return json.Unmarshal(schema["type"], &schemaType) == nil && schemaType == "object"
}
