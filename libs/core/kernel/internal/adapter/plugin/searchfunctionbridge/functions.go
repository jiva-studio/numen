package searchfunctionbridge

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
)

// Functions calls the plugins that answer query functions. Each call holds a use lease on its plugin for
// its length.
type Functions struct {
	activator bridge.Activator
}

// Open makes the bridge over the plugins the activator starts. It starts none.
func Open(activator bridge.Activator) Functions {
	return Functions{activator: activator}
}

// MatchDocuments returns the paths among paths the call holds for.
func (f Functions) MatchDocuments(ctx context.Context, call searchquery.FunctionCall, paths []string) (_ searchquery.FunctionAnswer, err error) {
	client, lease, err := f.open(ctx, call)
	if err != nil {
		return searchquery.FunctionAnswer{}, err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	res, err := client.MatchDocuments(ctx, connect.NewRequest(&v1.MatchDocumentsRequest{Call: writeCall(call), Paths: paths}))
	if err != nil {
		return searchquery.FunctionAnswer{}, readError(ctx, call, err)
	}
	return searchquery.FunctionAnswer{Paths: res.Msg.GetPaths(), Warning: res.Msg.GetWarning()}, nil
}

// FindDocuments returns at most limit paths the call holds for and an estimate of their count.
func (f Functions) FindDocuments(ctx context.Context, call searchquery.FunctionCall, limit int) (_ searchquery.FunctionAnswer, err error) {
	client, lease, err := f.open(ctx, call)
	if err != nil {
		return searchquery.FunctionAnswer{}, err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	res, err := client.FindDocuments(ctx, connect.NewRequest(&v1.FindDocumentsRequest{Call: writeCall(call), Limit: int32(min(limit, maxLimit))}))
	if err != nil {
		return searchquery.FunctionAnswer{}, readError(ctx, call, err)
	}
	paths := res.Msg.GetPaths()
	if len(paths) > limit {
		paths = paths[:limit]
	}
	return searchquery.FunctionAnswer{
		Paths: paths, EstimatedCount: max(res.Msg.GetEstimatedCount(), int64(len(paths))), Warning: res.Msg.GetWarning(),
	}, nil
}

const maxLimit = 1 << 30

func (f Functions) open(ctx context.Context, call searchquery.FunctionCall) (numenv1connect.SearchFunctionServiceClient, *bridge.Lease, error) {
	client, lease, err := bridge.Open(ctx, f.activator, call.PluginID, "search function "+call.Function, numenv1connect.NewSearchFunctionServiceClient)
	if err != nil {
		return nil, nil, fmt.Errorf("start plugin %s: %w", call.PluginID, err)
	}
	return client, lease, nil
}

func writeCall(call searchquery.FunctionCall) *v1.SearchFunctionCall {
	wire := &v1.SearchFunctionCall{Function: call.Function, VaultId: call.VaultID, IndexGeneration: call.IndexGeneration}
	for _, arg := range call.Arguments {
		one := &v1.SearchFunctionArgument{Name: arg.Name}
		switch arg.Type {
		case searchquery.TypeInt:
			one.Value = &v1.SearchFunctionArgument_IntValue{IntValue: arg.Int}
		case searchquery.TypeBool:
			one.Value = &v1.SearchFunctionArgument_BoolValue{BoolValue: arg.Bool}
		case searchquery.TypeString, searchquery.TypeStringList, searchquery.TypeSearchResults,
			searchquery.TypeTimestamp, searchquery.TypeDyn:
			one.Value = &v1.SearchFunctionArgument_StringValue{StringValue: arg.String}
		}
		wire.Arguments = append(wire.Arguments, one)
	}
	return wire
}

func readError(ctx context.Context, call searchquery.FunctionCall, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if connect.CodeOf(err) == connect.CodeInvalidArgument {
		message := err.Error()
		if failure, isConnect := errors.AsType[*connect.Error](err); isConnect {
			message = failure.Message()
		}
		return fmt.Errorf("%w: %s", searchquery.ErrInvalidFunctionArguments, message)
	}
	return fmt.Errorf("plugin %s: %w", call.PluginID, err)
}
