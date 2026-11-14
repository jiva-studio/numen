// Package tools lets a plugin offer an agent tools: it lists them, runs them, and answers
// PluginToolService for the host.
//
// A tool is a function from an argument struct to a result struct. The schemas
// presented to an agent are derived from the two structs.
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

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
)

// Effect is what a tool does to the vault. It decides the permission the plugin must hold.
type Effect int

const (
	// ReadOnly is a tool that reads and changes nothing. The plugin needs vault:read.
	ReadOnly Effect = iota + 1
	// ReadWrite is a tool that changes the vault. The plugin needs vault:write.
	ReadWrite
)

// Spec describes a tool: its name, its purpose and where it is offered.
//
//numen:api
type Spec struct {
	// Name is what an agent calls the tool by.
	Name string
	// Title is the tool's display name.
	Title string
	// Description is what the tool is for and when an agent uses it.
	Description string
	Effect      Effect
	// Instructions are the sentences an agent reads about the tool before it calls it, each one
	// sentence. A sentence about several tools is attached to each of them with identical text; the
	// host lists it once.
	Instructions []string
	// Requires are the permissions the caller must have besides the one of the effect, such as
	// vault:manage. The host lists and runs the tool only for a caller that has all of them.
	Requires []string
}

// ErrDuplicate is a second tool added under a name.
var ErrDuplicate = errors.New("a tool of that name is already added")

// Registry is the tools one plugin offers. It serves PluginToolService.
type Registry struct {
	mu    sync.RWMutex
	tools []registered
}

type registered struct {
	tool *v1.PluginTool
	run  func(ctx context.Context, arguments []byte) *v1.ToolResult
}

func New() *Registry {
	return &Registry{}
}

// Add adds the tool that run implements.
// Schemas are derived from In and Out; an Out that is not a struct is returned as text.
// Arguments are validated against the schema of In before run is called, and a
// mismatch is rejected with an error result naming the argument.
// The text of every error result starts with the tool name and ": ".
func Add[In, Out any](r *Registry, spec Spec, run func(ctx context.Context, in In) (Out, error)) error {
	inputSchema, err := createSchema(reflect.TypeFor[In]())
	if err != nil {
		return fmt.Errorf("tool %s: arguments: %w", spec.Name, err)
	}
	input, err := json.Marshal(inputSchema)
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
		tool: &v1.PluginTool{
			Name:         spec.Name,
			Title:        spec.Title,
			Description:  spec.Description,
			InputSchema:  input,
			OutputSchema: output,
			Effect:       writeEffect(spec.Effect),
			Instructions: slices.Clone(spec.Instructions),
			Requires:     slices.Clone(spec.Requires),
		},
		run: func(ctx context.Context, arguments []byte) *v1.ToolResult {
			in, err := readArguments[In](inputSchema, arguments)
			if err != nil {
				return &v1.ToolResult{Text: spec.Name + ": " + err.Error(), IsError: true}
			}
			out, err := run(ctx, in)
			if err != nil {
				return &v1.ToolResult{Text: spec.Name + ": " + err.Error(), IsError: true}
			}
			return writeResult(out)
		},
	})
}

func (r *Registry) add(one registered) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if slices.ContainsFunc(r.tools, func(reg registered) bool { return reg.tool.GetName() == one.tool.GetName() }) {
		return fmt.Errorf("%w: %s", ErrDuplicate, one.tool.GetName())
	}
	r.tools = append(r.tools, one)
	return nil
}

// Mount registers the registry as PluginToolService on s.
func (r *Registry) Mount(s *server.Server) {
	s.Register(numenv1connect.NewPluginToolServiceHandler(registryHandler{registry: r}, s.HandlerOptions()...))
}

// registryHandler serves a Registry as PluginToolService.
type registryHandler struct {
	numenv1connect.UnimplementedPluginToolServiceHandler

	registry *Registry
}

// ListTools lists the tools in the order they were added.
func (h registryHandler) ListTools(
	context.Context, *connect.Request[v1.ListToolsRequest],
) (*connect.Response[v1.ListToolsResponse], error) {
	r := h.registry
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]*v1.PluginTool, 0, len(r.tools))
	for _, one := range r.tools {
		items = append(items, one.tool)
	}
	return connect.NewResponse(&v1.ListToolsResponse{Tools: items}), nil
}

// CallTool runs the tool named. A name nothing was added under, and arguments that are no JSON
// object, are answered with an error code.
func (h registryHandler) CallTool(
	ctx context.Context, req *connect.Request[v1.CallToolRequest],
) (*connect.Response[v1.CallToolResponse], error) {
	r := h.registry
	r.mu.RLock()
	index := slices.IndexFunc(r.tools, func(reg registered) bool { return reg.tool.GetName() == req.Msg.GetName() })
	var one registered
	if index >= 0 {
		one = r.tools[index]
	}
	r.mu.RUnlock()
	if index < 0 {
		return connect.NewResponse(&v1.CallToolResponse{Error: v1.ToolErrorCode_TOOL_ERROR_CODE_UNKNOWN_TOOL}), nil
	}
	if arguments := req.Msg.GetArguments(); len(arguments) > 0 && !isObject(arguments) {
		return connect.NewResponse(&v1.CallToolResponse{Error: v1.ToolErrorCode_TOOL_ERROR_CODE_BAD_ARGUMENTS}), nil
	}
	ctx = context.WithValue(ctx, callContextKey{}, req.Msg.GetContext())
	ctx = context.WithValue(ctx, vaultKey{}, req.Msg.GetVaultId())
	return connect.NewResponse(&v1.CallToolResponse{Result: one.run(ctx, req.Msg.GetArguments())}), nil
}

type callContextKey struct{}

type vaultKey struct{}

// GetCallContext returns the context of the query the call was made under: what
// the originating view shows, under keys that start with the id of the plugin
// that wrote them and a slash. It is empty for a query with none. A tool reads
// it as data about the call; it is not part of the arguments.
func GetCallContext(ctx context.Context) map[string]string {
	if request, ok := ctx.Value(callContextKey{}).(map[string]string); ok {
		return request
	}
	return nil
}

// GetVault is the identity of the vault the call works: the one the agent run was started on, or for
// a call from a program outside Numen the one the host shows. It is empty where there is none.
func GetVault(ctx context.Context) string {
	if vault, ok := ctx.Value(vaultKey{}).(string); ok {
		return vault
	}
	return ""
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

func createSchema(t reflect.Type) (*jsonschema.Schema, error) {
	if !isStruct(t) {
		return &jsonschema.Schema{Type: typeObject}, nil
	}
	schema, err := jsonschema.ForType(t, &jsonschema.ForOptions{})
	if err != nil {
		return nil, err
	}
	schema.Type, schema.Types = typeObject, nil
	return schema, nil
}

func schemaOf(t reflect.Type) ([]byte, error) {
	schema, err := createSchema(t)
	if err != nil {
		return nil, err
	}
	return json.Marshal(schema)
}

func writeResult(out any) *v1.ToolResult {
	encoded, err := json.Marshal(out)
	if err != nil {
		return &v1.ToolResult{Text: fmt.Sprintf("the result cannot be encoded: %v", err), IsError: true}
	}
	if isObject(encoded) {
		return &v1.ToolResult{Structured: encoded}
	}
	return &v1.ToolResult{Text: string(encoded)}
}

func (r *Registry) ListTools(ctx context.Context) (Listing, error) {
	res, err := registryHandler{registry: r}.ListTools(ctx, connect.NewRequest(&v1.ListToolsRequest{}))
	// gomutants:disable-next-line reason="the error branch runs only when the call fails, and no test makes it fail"
	if err != nil {
		return Listing{}, err
	}
	return readListing(res.Msg), nil
}

func (r *Registry) InvokeTool(ctx context.Context, call Call) (*CallResponse, error) {
	res, err := registryHandler{registry: r}.CallTool(ctx, connect.NewRequest(&v1.CallToolRequest{
		Name:      call.Name,
		Arguments: call.Arguments,
		Context:   call.Context,
		VaultId:   call.VaultID,
	}))
	if err != nil {
		return nil, err
	}
	return readCallResponse(res.Msg), nil
}
