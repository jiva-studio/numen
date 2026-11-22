package command

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// ListVersionItems lists every text version of the file over the shown vault. A file whose versions cannot
// be listed is answered with the code and message of the error.
func (h *Handler) ListVersionItems(
	ctx context.Context, r *connect.Request[v1.ListVersionItemsRequest],
) (*connect.Response[v1.ListVersionItemsResponse], error) {
	lists := h.listItems.Execute(ctx, h.vaults.GetCurrentVault(), r.Msg.GetPath(), []plugincommand.CommandID{{Command: text.DeleteCommand}})
	if err := ctx.Err(); err != nil {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}
	list := newWireItemList(lists[0])
	return connect.NewResponse(&v1.ListVersionItemsResponse{Items: list.GetItems(), Error: list.GetError(), Message: list.GetMessage()}), nil
}

// DeleteVersion deletes the text version of the file over the shown vault. A version the file does not offer
// is answered with an error code and a user-readable message.
func (h *Handler) DeleteVersion(
	ctx context.Context, r *connect.Request[v1.DeleteVersionRequest],
) (*connect.Response[v1.DeleteVersionResponse], error) {
	_, err := h.runCommand.Execute(ctx, h.vaults.GetCurrentVault(), plugin.CommandCall{
		Command: text.DeleteCommand, Path: r.Msg.GetPath(), ItemID: r.Msg.GetVersionId(),
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, connect.NewError(connect.CodeCanceled, ctxErr)
		}
		return connect.NewResponse(&v1.DeleteVersionResponse{Error: getErrorCode(err), Message: err.Error()}), nil
	}
	return connect.NewResponse(&v1.DeleteVersionResponse{}), nil
}
