// Package exposetools exposes the tools of the host to MCP clients: it keeps the tools registered
// on an MCP server equal to the host's listing and forwards tool calls to the host.
package exposetools

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-mcp-server/backend/mcptool"
)

// Host lists the tools of the host and calls them.
type Host interface {
	ListTools(ctx context.Context) (tools.Listing, error)
	InvokeTool(ctx context.Context, call tools.Call) (*tools.CallResponse, error)
}

// Registry holds the tools an MCP server offers.
type Registry interface {
	AddTool(tool *sdk.Tool, handler sdk.ToolHandler)
	RemoveTools(names ...string)
}

// Aggregator discovers tools from the host and registers them on an MCP server.
type Aggregator struct {
	clientMu   sync.RWMutex
	client     Host
	registry   Registry
	mu         sync.RWMutex
	registered map[string]*sdk.Tool
	syncMu     sync.Mutex
	sentences  []string
}

// NewAggregator creates an aggregator for the host and the registry. Both may be nil: without a
// host the aggregator registers nothing and refuses calls.
func NewAggregator(client Host, registry Registry) *Aggregator {
	return &Aggregator{
		clientMu:   sync.RWMutex{},
		client:     client,
		registry:   registry,
		mu:         sync.RWMutex{},
		syncMu:     sync.Mutex{},
		registered: make(map[string]*sdk.Tool),
		sentences:  nil,
	}
}

// HasClient reports whether the aggregator has a host to ask.
func (a *Aggregator) HasClient() bool {
	return a.getClient() != nil
}

// SetClient replaces the host the aggregator asks.
func (a *Aggregator) SetClient(client Host) {
	a.clientMu.Lock()
	defer a.clientMu.Unlock()
	a.client = client
}

func (a *Aggregator) getClient() Host {
	a.clientMu.RLock()
	defer a.clientMu.RUnlock()
	return a.client
}

// GetRegisteredTools returns the sorted list of currently registered tool names.
func (a *Aggregator) GetRegisteredTools() []string {
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
func (a *Aggregator) GetInstructions() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return strings.Join(a.sentences, " ")
}

// SyncTools queries the host for available tools and updates the registry.
func (a *Aggregator) SyncTools(ctx context.Context) error {
	client := a.getClient()
	if client == nil || a.registry == nil {
		return nil
	}
	a.syncMu.Lock()
	defer a.syncMu.Unlock()

	listing, err := client.ListTools(ctx)
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

func readDeclarations(listing tools.Listing) (map[string]*sdk.Tool, []string) {
	declared := make(map[string]*sdk.Tool)
	var sentences []string
	for _, wireTool := range listing.Tools {
		if wireTool.GetName() == "" {
			continue
		}
		declared[wireTool.GetName()] = mcptool.NewTool(wireTool)
		for _, sentence := range wireTool.GetInstructions() {
			if !slices.Contains(sentences, sentence) {
				sentences = append(sentences, sentence)
			}
		}
	}
	return declared, sentences
}

func (a *Aggregator) updateTools(declared map[string]*sdk.Tool) {
	for name, tool := range declared {
		if previous, isRegistered := a.registered[name]; isRegistered && reflect.DeepEqual(previous, tool) {
			continue
		}
		a.registry.AddTool(tool, func(callCtx context.Context, req *sdk.CallToolRequest) (*sdk.CallToolResult, error) {
			var args []byte
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
		a.registry.RemoveTools(toRemove...)
	}
}

// ExecuteTool forwards a tool call to the host and translates the response. The host runs it in the
// vault it has open, without request context: the call does not originate from a Numen request.
func (a *Aggregator) ExecuteTool(ctx context.Context, name string, args []byte) (*sdk.CallToolResult, error) {
	client := a.getClient()
	if client == nil {
		return mcptool.NewErrorResult("no host client configured"), nil
	}

	res, err := client.InvokeTool(ctx, tools.Call{
		Name:      name,
		Arguments: args,
		Context:   nil,
		VaultID:   "",
	})
	if err != nil {
		return nil, fmt.Errorf("call host tool %s: %w", name, err)
	}
	return mcptool.NewResult(name, res), nil
}
