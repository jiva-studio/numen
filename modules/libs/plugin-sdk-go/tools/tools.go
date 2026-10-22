// Package tools lets a plugin offer an agent tools: it lists them, runs them, and answers
// PluginToolService for the host.
//
// A tool is a function of a struct of arguments that answers with a struct. The schemas an agent is
// told are derived from the two structs.
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"

	"connectrpc.com/connect"
	"github.com/google/jsonschema-go/jsonschema"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"
)

// Effect is what a tool does to the vault. It decides the permission the plugin must hold.
type Effect int

const (
	// Reads is a tool that reads and changes nothing. The plugin needs vault:read.
	Reads Effect = iota + 1
	// Writes is a tool that changes the vault. The plugin needs vault:write.
	Writes
)

// Step is how a call of a tool is shown to a person.
type Step int

const (
	// ByEffect shows a tool that reads as a look and one that writes as an edit.
	ByEffect Step = iota
	// Search shows a call as a search for notes.
	Search
	// Remove shows a call as taking something out of the vault.
	Remove
	// Move shows a call as filing something elsewhere or under another name.
	Move
)

// Server is one of the servers the host builds for an agent.
type Server int

const (
	// Full is the server an agent works the whole vault through.
	Full Server = iota + 1
	// Reading is the server an agent answers from, whose every tool reads.
	Reading
	// Review is the server of the window a person reviews cards in.
	Review
)

// Spec says what a tool is called, what it is for and where it is offered.
type Spec struct {
	// Name is what an agent calls the tool by.
	Name string
	// Title is what a person reads the tool as.
	Title string
	// Description is what the tool is for and when an agent reaches for it.
	Description string
	Effect      Effect
	// Step is how a call is shown to a person.
	Step Step
	// Servers are the servers the tool is offered on. None is the full server alone.
	Servers []Server
}

// ErrDuplicate is a second tool added under a name.
var ErrDuplicate = errors.New("a tool of that name is already added")

// Registry is the tools one plugin offers. It serves PluginToolService.
type Registry struct {
	numenv1connect.UnimplementedPluginToolServiceHandler

	mu    sync.RWMutex
	tools []registered
}

type registered struct {
	listed *numenv1.PluginTool
	run    func(ctx context.Context, arguments []byte) *numenv1.ToolResult
}

// New makes a registry of no tools.
func New() *Registry {
	return &Registry{}
}

// Add adds the tool run answers for. The schemas of the arguments and of the result are derived
// from In and Out; an Out that is not a struct is answered as text. A run that returns an error is
func Add[In, Out any](r *Registry, spec Spec, run func(ctx context.Context, in In) (Out, error)) error {
	input, err := schemaOf(reflect.TypeFor[In]())
	if err != nil {
		return fmt.Errorf("tool %s: arguments: %w", spec.Name, err)
	}
	var output []byte
	if isStruct(reflect.TypeFor[Out]()) {
		if output, err = schemaOf(reflect.TypeFor[Out]()); err != nil {
			return fmt.Errorf("tool %s: result: %w", spec.Name, err)
		}
	}
	return r.add(registered{
		listed: &numenv1.PluginTool{
			Name:         spec.Name,
			Title:        spec.Title,
			Description:  spec.Description,
			InputSchema:  input,
			OutputSchema: output,
			Effect:       writeEffect(spec.Effect),
			Servers:      writeServers(spec.Servers),
			Step:         writeStep(spec.Step),
		},
		run: func(ctx context.Context, arguments []byte) *numenv1.ToolResult {
			var in In
			if len(arguments) > 0 {
				if err := json.Unmarshal(arguments, &in); err != nil {
					return &numenv1.ToolResult{Text: fmt.Sprintf("the arguments do not fit: %v", err), IsRefusal: true}
				}
			}
			out, err := run(ctx, in)
			if err != nil {
				return &numenv1.ToolResult{Text: err.Error(), IsRefusal: true}
			}
			return writeResult(out)
		},
	})
}

func (r *Registry) add(one registered) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if slices.ContainsFunc(r.tools, func(held registered) bool { return held.listed.GetName() == one.listed.GetName() }) {
		return fmt.Errorf("%w: %s", ErrDuplicate, one.listed.GetName())
	}
	r.tools = append(r.tools, one)
	return nil
}

// Mount registers the registry as PluginToolService on s.
func (r *Registry) Mount(s *server.Server) {
	s.Register(numenv1connect.NewPluginToolServiceHandler(r, s.HandlerOptions()...))
}

// ListTools lists the tools in the order they were added.
func (r *Registry) ListTools(
	context.Context, *connect.Request[numenv1.ListToolsRequest],
) (*connect.Response[numenv1.ListToolsResponse], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]*numenv1.PluginTool, 0, len(r.tools))
	for _, one := range r.tools {
		items = append(items, one.listed)
	}
	return connect.NewResponse(&numenv1.ListToolsResponse{Tools: items}), nil
}

// CallTool runs the tool named. A name nothing was added under, and arguments that are no JSON
// object, are answered with an error code.
func (r *Registry) CallTool(
	ctx context.Context, req *connect.Request[numenv1.CallToolRequest],
) (*connect.Response[numenv1.CallToolResponse], error) {
	r.mu.RLock()
	index := slices.IndexFunc(r.tools, func(held registered) bool { return held.listed.GetName() == req.Msg.GetName() })
	var one registered
	if index >= 0 {
		one = r.tools[index]
	}
	r.mu.RUnlock()
	if index < 0 {
		return connect.NewResponse(&numenv1.CallToolResponse{Error: numenv1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL}), nil
	}
	if arguments := req.Msg.GetArguments(); len(arguments) > 0 && !isObject(arguments) {
		return connect.NewResponse(&numenv1.CallToolResponse{Error: numenv1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS}), nil
	}
	ctx = context.WithValue(ctx, askedKey{}, req.Msg.GetContext())
	return connect.NewResponse(&numenv1.CallToolResponse{Result: one.run(ctx, req.Msg.GetArguments())}), nil
}

type askedKey struct{}

// Asked is what the person is looking at in the window the agent was started from, as the host names
// it: focus, mark and face for the window cards are reviewed in. It is empty for a call made from a
// window that shows the person nothing in particular. A tool reads it as data about the call, and it
// is no part of the arguments.
func Asked(ctx context.Context) map[string]string {
	request, _ := ctx.Value(askedKey{}).(map[string]string)
	return request
}

func isObject(raw []byte) bool {
	var object map[string]json.RawMessage
	return json.Unmarshal(raw, &object) == nil
}

func isStruct(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct
}

// schemaOf is the JSON schema of a struct as a JSON object, and an empty object schema for any
// other type.
func schemaOf(t reflect.Type) ([]byte, error) {
	if !isStruct(t) {
		return []byte(`{"type":"object"}`), nil
	}
	schema, err := jsonschema.ForType(t, &jsonschema.ForOptions{})
	if err != nil {
		return nil, err
	}
	return json.Marshal(schema)
}

// writeResult is what a tool answered, as a structured result where it is an object and as text
// where it is not.
func writeResult(out any) *numenv1.ToolResult {
	encoded, err := json.Marshal(out)
	if err != nil {
		return &numenv1.ToolResult{Text: fmt.Sprintf("the result cannot be encoded: %v", err), IsRefusal: true}
	}
	if isObject(encoded) {
		return &numenv1.ToolResult{Structured: encoded}
	}
	return &numenv1.ToolResult{Text: string(encoded)}
}

func writeEffect(effect Effect) numenv1.ToolEffect {
	switch effect {
	case Reads:
		return numenv1.ToolEffect_TOOL_EFFECT_READS
	case Writes:
		return numenv1.ToolEffect_TOOL_EFFECT_WRITES
	}
	return numenv1.ToolEffect_TOOL_EFFECT_UNSPECIFIED
}

func writeStep(step Step) numenv1.ToolStep {
	switch step {
	case Search:
		return numenv1.ToolStep_TOOL_STEP_SEARCH
	case Remove:
		return numenv1.ToolStep_TOOL_STEP_REMOVE
	case Move:
		return numenv1.ToolStep_TOOL_STEP_MOVE
	case ByEffect:
	}
	return numenv1.ToolStep_TOOL_STEP_UNSPECIFIED
}

func writeServers(servers []Server) []numenv1.ToolServer {
	wire := make([]numenv1.ToolServer, 0, len(servers))
	for _, one := range servers {
		switch one {
		case Full:
			wire = append(wire, numenv1.ToolServer_TOOL_SERVER_FULL)
		case Reading:
			wire = append(wire, numenv1.ToolServer_TOOL_SERVER_READING)
		case Review:
			wire = append(wire, numenv1.ToolServer_TOOL_SERVER_REVIEW)
		}
	}
	return wire
}
