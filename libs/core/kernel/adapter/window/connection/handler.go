// Package connection serves a window's ProviderConnectionService: the connections to services and
// programs, and the kinds of connection the installed plugins provide.
package connection

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/connection"
	connections "github.com/jiva-studio/numen/libs/core/kernel/usecase/connection"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/rowoptions"
)

// UseCases are the use cases ProviderConnectionService is answered through.
type UseCases struct {
	Create    *connections.Create
	Update    *connections.Update
	Delete    *connections.Delete
	List      *connections.List
	ListTypes *connections.ListTypes
	// ListOperationTitles lists the titles of the operations; nil answers no title.
	ListOperationTitles *connections.ListOperationTitles
	// RowOptions lists the values of row fields; nil answers every list with no options.
	RowOptions *rowoptions.Options
}

// Handler serves ProviderConnectionService over the use cases on connections.
type Handler struct {
	connections UseCases
}

// NewHandler is the handler over the connections of this installation.
func NewHandler(useCases UseCases) *Handler {
	return &Handler{connections: useCases}
}

// ListProviderConnectionTypes returns the kinds of connection the plugins declare, and the titles the
// plugins give their operations.
func (h *Handler) ListProviderConnectionTypes(
	_ context.Context, _ *connect.Request[v1.ListProviderConnectionTypesRequest],
) (*connect.Response[v1.ListProviderConnectionTypesResponse], error) {
	response := &v1.ListProviderConnectionTypesResponse{}
	for _, one := range h.connections.ListTypes.Execute() {
		wire, err := writeType(one)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		response.Types = append(response.Types, wire)
	}
	if h.connections.ListOperationTitles != nil {
		for _, one := range h.connections.ListOperationTitles.Execute() {
			response.Operations = append(response.Operations, &v1.ManifestOperation{Name: one.Name, Title: one.Title})
		}
	}
	return connect.NewResponse(response), nil
}

// ListProviderConnections returns the connections in list order.
func (h *Handler) ListProviderConnections(
	ctx context.Context, _ *connect.Request[v1.ListProviderConnectionsRequest],
) (*connect.Response[v1.ListProviderConnectionsResponse], error) {
	entries, err := h.connections.List.Execute(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &v1.ListProviderConnectionsResponse{}
	for _, one := range entries {
		wire, err := writeConnection(one)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		response.Connections = append(response.Connections, wire)
	}
	return connect.NewResponse(response), nil
}

// CreateProviderConnection adds a connection and keeps its secret fields in the key store.
func (h *Handler) CreateProviderConnection(
	ctx context.Context, r *connect.Request[v1.CreateProviderConnectionRequest],
) (*connect.Response[v1.CreateProviderConnectionResponse], error) {
	entry, err := h.connections.Create.Execute(ctx, connections.CreateRequest{
		TypeID:  r.Msg.GetTypeId(),
		Name:    r.Msg.GetName(),
		Fields:  readFields(r.Msg.GetFields()),
		Secrets: r.Msg.GetSecrets(),
	})
	if err != nil {
		code, served := getErrorCode(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.CreateProviderConnectionResponse{Error: code, Detail: getDetail(err)}), nil
	}
	wire, err := writeConnection(entry)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.CreateProviderConnectionResponse{Connection: wire}), nil
}

// UpdateProviderConnection changes the connection that has the revision the request carries.
func (h *Handler) UpdateProviderConnection(
	ctx context.Context, r *connect.Request[v1.UpdateProviderConnectionRequest],
) (*connect.Response[v1.UpdateProviderConnectionResponse], error) {
	entry, err := h.connections.Update.Execute(ctx, connections.UpdateRequest{
		ID:       r.Msg.GetId(),
		Revision: int(r.Msg.GetRevision()),
		Name:     r.Msg.GetName(),
		Fields:   readFields(r.Msg.GetFields()),
		Secrets:  r.Msg.GetSecrets(),
	})
	if err != nil {
		code, served := getErrorCode(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.UpdateProviderConnectionResponse{Error: code, Detail: getDetail(err)}), nil
	}
	wire, err := writeConnection(entry)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.UpdateProviderConnectionResponse{Connection: wire}), nil
}

// DeleteProviderConnection removes a connection and its secret fields.
func (h *Handler) DeleteProviderConnection(
	ctx context.Context, r *connect.Request[v1.DeleteProviderConnectionRequest],
) (*connect.Response[v1.DeleteProviderConnectionResponse], error) {
	removed, err := h.connections.Delete.Execute(ctx, r.Msg.GetId())
	if err != nil {
		code, served := getErrorCode(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.DeleteProviderConnectionResponse{Error: code, Detail: getDetail(err)}), nil
	}
	return connect.NewResponse(&v1.DeleteProviderConnectionResponse{RemovedUses: writeUses(removed)}), nil
}

// ListRowOptions returns the values a row field of the type of a connection can take, as its provider
// lists them.
func (h *Handler) ListRowOptions(
	ctx context.Context, r *connect.Request[v1.ListRowOptionsRequest],
) (*connect.Response[v1.ListRowOptionsResponse], error) {
	if h.connections.RowOptions == nil {
		return connect.NewResponse(&v1.ListRowOptionsResponse{}), nil
	}
	listed, err := h.connections.RowOptions.ListOptions(ctx, r.Msg.GetConnectionId(), r.Msg.GetField(), r.Msg.GetOptions(), r.Msg.GetOperations())
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, connect.NewError(connect.CodeCanceled, ctxErr)
		}
		code, served := getErrorCode(err)
		if !served {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.ListRowOptionsResponse{Error: code, Detail: getDetail(err)}), nil
	}
	options := make([]*v1.Option, len(listed))
	for i, one := range listed {
		options[i] = &v1.Option{Value: one.Value, Title: one.Title, Group: one.Group}
	}
	return connect.NewResponse(&v1.ListRowOptionsResponse{Options: options}), nil
}

func getDetail(err error) string {
	sentinels := []error{
		connection.ErrNotFound, connections.ErrNameTaken, connections.ErrInvalidField,
		connection.ErrTypeUnavailable, connection.ErrKeyStore, connection.ErrStaleRevision,
		rowoptions.ErrProviderUnavailable, rowoptions.ErrNoOptions,
	}
	for _, sentinel := range sentinels {
		if errors.Is(err, sentinel) {
			return strings.TrimPrefix(err.Error(), sentinel.Error()+": ")
		}
	}
	return err.Error()
}

func getErrorCode(err error) (v1.ProviderConnectionErrorCode, bool) {
	switch {
	case errors.Is(err, connection.ErrNotFound):
		return v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_NOT_FOUND, true
	case errors.Is(err, connections.ErrNameTaken):
		return v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_NAME_TAKEN, true
	case errors.Is(err, connections.ErrInvalidField):
		return v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_INVALID_FIELD, true
	case errors.Is(err, connection.ErrTypeUnavailable):
		return v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_TYPE_UNAVAILABLE, true
	case errors.Is(err, connection.ErrKeyStore):
		return v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_KEY_STORE_FAILED, true
	case errors.Is(err, connection.ErrStaleRevision):
		return v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_STALE_REVISION, true
	case errors.Is(err, rowoptions.ErrProviderUnavailable):
		return v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_PROVIDER_UNAVAILABLE, true
	case errors.Is(err, rowoptions.ErrNoOptions):
		return v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_INVALID_FIELD, true
	}
	return v1.ProviderConnectionErrorCode_PROVIDER_CONNECTION_ERROR_CODE_UNSPECIFIED, false
}
