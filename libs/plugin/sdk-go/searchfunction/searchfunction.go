// Package searchfunction lets a plugin answer the query functions it declares in contributes.searchFunctions,
// over SearchFunctionService. A function answers for the documents of one vault, named by their paths.
package searchfunction

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"connectrpc.com/connect"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
)

// Call is one use of a function in a query: the vault it searches and the generation of the index the
// search runs on, which a plugin shows with an answer from a cache that may be behind it.
//
//numen:api
type Call struct {
	VaultID         string
	IndexGeneration string
	Arguments       Arguments
}

// Arguments are the values of a call by the names the manifest gives them.
type Arguments struct {
	values []*v1.SearchFunctionArgument
}

func (a Arguments) find(name string) (*v1.SearchFunctionArgument, bool) {
	index := slices.IndexFunc(a.values, func(one *v1.SearchFunctionArgument) bool { return one.GetName() == name })
	if index < 0 {
		return nil, false
	}
	return a.values[index], true
}

// GetString returns the argument of the name, and false when the call has no string under it.
func (a Arguments) GetString(name string) (string, bool) {
	one, isFound := a.find(name)
	if !isFound {
		return "", false
	}
	value, isString := one.GetValue().(*v1.SearchFunctionArgument_StringValue)
	if !isString {
		return "", false
	}
	return value.StringValue, true
}

// GetInt returns the argument of the name, and false when the call has no int under it.
func (a Arguments) GetInt(name string) (int64, bool) {
	one, isFound := a.find(name)
	if !isFound {
		return 0, false
	}
	value, isInt := one.GetValue().(*v1.SearchFunctionArgument_IntValue)
	if !isInt {
		return 0, false
	}
	return value.IntValue, true
}

// GetBool returns the argument of the name, and false when the call has no bool under it.
func (a Arguments) GetBool(name string) (bool, bool) {
	one, isFound := a.find(name)
	if !isFound {
		return false, false
	}
	value, isBool := one.GetValue().(*v1.SearchFunctionArgument_BoolValue)
	if !isBool {
		return false, false
	}
	return value.BoolValue, true
}

// MatchResult are the paths among those asked about for which the function holds. Warning is a text the query
// shows with its results, such as an answer that may be out of date.
//
//numen:api
type MatchResult struct {
	Paths   []string
	Warning string
}

// Found are at most limit paths the function holds for and how many it holds for in all.
//
//numen:api
type Found struct {
	Paths          []string
	EstimatedCount int64
	Warning        string
}

// Function answers one declared function: Match tells which of the paths it holds for, Find lists the paths
// it holds for.
//
//numen:api
type Function struct {
	Match func(ctx context.Context, call Call, paths []string) (MatchResult, error)
	Find  func(ctx context.Context, call Call, limit int) (Found, error)
}

var (
	// ErrDuplicate is a second function added under a name.
	ErrDuplicate = errors.New("a function of that name is already added")
	// ErrInvalidArguments, wrapped in an error of Match or Find, answers that the arguments of the call are
	// wrong; the query shows its message at the call.
	ErrInvalidArguments = errors.New("invalid arguments")
)

// Registry is the functions one plugin answers. It serves SearchFunctionService.
type Registry struct {
	mu        sync.RWMutex
	functions map[string]Function
}

func New() *Registry {
	return &Registry{functions: map[string]Function{}}
}

// Add registers f under the name a manifest declares; a second function of one name is ErrDuplicate.
func (r *Registry) Add(name string, f Function) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, isAdded := r.functions[name]; isAdded {
		return fmt.Errorf("%w: %s", ErrDuplicate, name)
	}
	r.functions[name] = f
	return nil
}

// Mount registers the registry as SearchFunctionService on s.
func (r *Registry) Mount(s *server.Server) {
	s.RegisterAuthenticated(numenv1connect.NewSearchFunctionServiceHandler(handler{registry: r}))
}

type handler struct {
	numenv1connect.UnimplementedSearchFunctionServiceHandler

	registry *Registry
}

func (h handler) get(wire *v1.SearchFunctionCall) (Function, Call, error) {
	h.registry.mu.RLock()
	f, isAdded := h.registry.functions[wire.GetFunction()]
	h.registry.mu.RUnlock()
	if !isAdded {
		return Function{}, Call{}, connect.NewError(connect.CodeNotFound, fmt.Errorf("no search function %q", wire.GetFunction()))
	}
	call := Call{VaultID: wire.GetVaultId(), IndexGeneration: wire.GetIndexGeneration(), Arguments: Arguments{values: wire.GetArguments()}}
	return f, call, nil
}

// MatchDocuments answers the paths of the request the function holds for; a path the function returns
// that the request does not hold is dropped.
func (h handler) MatchDocuments(
	ctx context.Context, req *connect.Request[v1.MatchDocumentsRequest],
) (*connect.Response[v1.MatchDocumentsResponse], error) {
	f, call, err := h.get(req.Msg.GetCall())
	if err != nil {
		return nil, err
	}
	if f.Match == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("the function does not match paths"))
	}
	asked := req.Msg.GetPaths()
	matched, err := f.Match(ctx, call, asked)
	if err != nil {
		return nil, writeError(err)
	}
	isAsked := make(map[string]bool, len(asked))
	for _, path := range asked {
		isAsked[path] = true
	}
	paths := slices.DeleteFunc(slices.Clone(matched.Paths), func(path string) bool { return !isAsked[path] })
	return connect.NewResponse(&v1.MatchDocumentsResponse{Paths: paths, Warning: matched.Warning}), nil
}

// FindDocuments answers at most the limit of paths the function holds for.
func (h handler) FindDocuments(
	ctx context.Context, req *connect.Request[v1.FindDocumentsRequest],
) (*connect.Response[v1.FindDocumentsResponse], error) {
	f, call, err := h.get(req.Msg.GetCall())
	if err != nil {
		return nil, err
	}
	if f.Find == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("the function does not find paths"))
	}
	limit := int(req.Msg.GetLimit())
	found, err := f.Find(ctx, call, limit)
	if err != nil {
		return nil, writeError(err)
	}
	paths := found.Paths[:min(limit, len(found.Paths))]
	return connect.NewResponse(&v1.FindDocumentsResponse{
		Paths: paths, EstimatedCount: max(found.EstimatedCount, int64(len(paths))), Warning: found.Warning,
	}), nil
}

func writeError(err error) error {
	if errors.Is(err, ErrInvalidArguments) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return err
}
