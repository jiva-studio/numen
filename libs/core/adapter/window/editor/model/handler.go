// Package model serves a window's ModelService: the models and the reasoning levels of a connection.
package model

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/usecase/modellist"
)

// Handler serves ModelService over the lists a provider gives.
type Handler struct {
	lists *modellist.Lists
}

// NewHandler is the handler over the lists of this installation.
func NewHandler(lists *modellist.Lists) *Handler { return &Handler{lists: lists} }

// ListModels returns the models of a connection that fit an operation.
func (h *Handler) ListModels(
	ctx context.Context, request *connect.Request[v1.ListModelsRequest],
) (*connect.Response[v1.ListModelsResponse], error) {
	models, err := h.lists.ListModels(ctx, request.Msg.GetConnectionId(), request.Msg.GetOperation())
	if err != nil {
		code, detail, isServed := readFailure(ctx, err)
		if !isServed {
			return nil, newStatusError(ctx, err)
		}
		return connect.NewResponse(&v1.ListModelsResponse{Error: code, Detail: detail}), nil
	}
	wire := make([]*v1.ConnectionModel, len(models))
	for i, one := range models {
		wire[i] = &v1.ConnectionModel{Id: one.ID, Title: one.Title, Operations: one.Operations}
	}
	return connect.NewResponse(&v1.ListModelsResponse{Models: wire}), nil
}

// ListReasoningLevels returns the reasoning levels of a model of a connection.
func (h *Handler) ListReasoningLevels(
	ctx context.Context, request *connect.Request[v1.ListReasoningLevelsRequest],
) (*connect.Response[v1.ListReasoningLevelsResponse], error) {
	levels, err := h.lists.ListReasoningLevels(ctx, request.Msg.GetConnectionId(), request.Msg.GetModel())
	if err != nil {
		code, detail, isServed := readFailure(ctx, err)
		if !isServed {
			return nil, newStatusError(ctx, err)
		}
		return connect.NewResponse(&v1.ListReasoningLevelsResponse{Error: code, Detail: detail}), nil
	}
	wire := make([]*v1.ConnectionReasoningLevel, len(levels))
	for i, one := range levels {
		wire[i] = &v1.ConnectionReasoningLevel{Value: one.Value, Title: one.Title}
	}
	return connect.NewResponse(&v1.ListReasoningLevelsResponse{Levels: wire}), nil
}

func readFailure(ctx context.Context, err error) (v1.ModelErrorCode, string, bool) {
	if ctx.Err() != nil {
		return v1.ModelErrorCode_MODEL_ERROR_CODE_UNSPECIFIED, "", false
	}
	code, isServed := getErrorCode(err)
	if !isServed {
		return code, "", false
	}
	if errors.Is(err, connection.ErrTypeUnavailable) {
		return code, "type unavailable", true
	}
	return code, strings.TrimPrefix(err.Error(), modellist.ErrProviderUnavailable.Error()+": "), true
}

func getErrorCode(err error) (v1.ModelErrorCode, bool) {
	switch {
	case errors.Is(err, connection.ErrNotFound):
		return v1.ModelErrorCode_MODEL_ERROR_CODE_NOT_FOUND, true
	case errors.Is(err, connection.ErrTypeUnavailable), errors.Is(err, modellist.ErrProviderUnavailable):
		return v1.ModelErrorCode_MODEL_ERROR_CODE_PROVIDER_UNAVAILABLE, true
	}
	return v1.ModelErrorCode_MODEL_ERROR_CODE_UNSPECIFIED, false
}

func newStatusError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return connect.NewError(connect.CodeCanceled, ctxErr)
	}
	return connect.NewError(connect.CodeInternal, err)
}
