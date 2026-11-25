package hostendpoint

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchrun"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	searchqueryusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/searchquery"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func (c searchCall) getRunErrorCode(needsVault bool) v1.ErrorCode {
	if c.search == nil || c.search.Runs == nil {
		return v1.ErrorCode_ERROR_CODE_NO_MODEL
	}
	return c.getRefusal(needsVault)
}

func answerRun(ctx context.Context, err error) (v1.ErrorCode, error) {
	if err == nil {
		return v1.ErrorCode_ERROR_CODE_UNSPECIFIED, nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return 0, connect.NewError(connect.CodeCanceled, ctxErr)
	}
	switch {
	case errors.Is(err, searchrun.ErrNotFound):
		return v1.ErrorCode_ERROR_CODE_NOT_FOUND, nil
	case errors.Is(err, searchrun.ErrTooManySearches):
		return v1.ErrorCode_ERROR_CODE_LIMIT_EXCEEDED, nil
	case errors.Is(err, searchrun.ErrInvalidPageToken):
		return v1.ErrorCode_ERROR_CODE_INVALID, nil
	}
	return 0, connect.NewError(connect.CodeInternal, err)
}

func (e *searchEndpoint) StartSearch(
	ctx context.Context, req *connect.Request[v1.StartSearchRequest],
) (*connect.Response[v1.StartSearchResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionIndexRead, true)
	if err != nil {
		return nil, err
	}
	if code := call.getRunErrorCode(true); code != 0 {
		return connect.NewResponse(&v1.StartSearchResponse{Error: code}), nil
	}
	id, err := call.search.Runs.Start.Execute(ctx, searchrun.Start{
		Owner: call.pluginID, Vault: call.vault, Query: req.Msg.GetQuery(),
		Priority: wire.ConvertPriorityFromWire(req.Msg.GetPriority()), IsBackground: req.Msg.GetIsBackground(),
	})
	code, err := answerRun(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.StartSearchResponse{Error: code}), err
	}
	return connect.NewResponse(&v1.StartSearchResponse{Id: string(id)}), nil
}

func (e *searchEndpoint) GetSearch(
	ctx context.Context, req *connect.Request[v1.GetSearchRequest],
) (*connect.Response[v1.GetSearchResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionIndexRead, false)
	if err != nil {
		return nil, err
	}
	if code := call.getRunErrorCode(false); code != 0 {
		return connect.NewResponse(&v1.GetSearchResponse{Error: code}), nil
	}
	page, err := call.search.Runs.Get.Execute(ctx, searchqueryusecase.Read{
		Owner: call.pluginID, ID: searchrun.ID(req.Msg.GetId()), Wait: time.Duration(max(req.Msg.GetWaitMs(), 0)) * time.Millisecond,
		PageSize: int(req.Msg.GetPageSize()), PageToken: req.Msg.GetPageToken(),
	})
	code, err := answerRun(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.GetSearchResponse{Error: code}), err
	}
	state, err := writeSearchState(page)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.GetSearchResponse{Search: state}), nil
}

func (e *searchEndpoint) DeleteSearch(
	ctx context.Context, req *connect.Request[v1.DeleteSearchRequest],
) (*connect.Response[v1.DeleteSearchResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionIndexRead, false)
	if err != nil {
		return nil, err
	}
	if code := call.getRunErrorCode(false); code != 0 {
		return connect.NewResponse(&v1.DeleteSearchResponse{Error: code}), nil
	}
	code, err := answerRun(ctx, call.search.Runs.Remove.Execute(ctx, call.pluginID, searchrun.ID(req.Msg.GetId())))
	return connect.NewResponse(&v1.DeleteSearchResponse{Error: code}), err
}

func (e *searchEndpoint) ListSearchFunctions(
	_ context.Context, req *connect.Request[v1.ListSearchFunctionsRequest],
) (*connect.Response[v1.ListSearchFunctionsResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionIndexRead, false)
	if err != nil {
		return nil, err
	}
	if code := call.getRunErrorCode(false); code != 0 {
		return connect.NewResponse(&v1.ListSearchFunctionsResponse{Error: code}), nil
	}
	catalog, revision := call.search.Runs.Functions.Execute()
	return connect.NewResponse(&v1.ListSearchFunctionsResponse{Functions: writeSearchFunctions(catalog), Revision: revision}), nil
}
