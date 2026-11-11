package command

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// ListCommandItems lists the items each command of the request offers over the path, one list per
// command in the order of the request. A command that is unknown, or whose items cannot be listed, has a
// list with the code and message of the error.
func (h *Handler) ListCommandItems(
	ctx context.Context, r *connect.Request[v1.ListCommandItemsRequest],
) (*connect.Response[v1.ListCommandItemsResponse], error) {
	commands := make([]plugincommand.CommandID, 0, len(r.Msg.GetCommands()))
	for _, one := range r.Msg.GetCommands() {
		commands = append(commands, plugincommand.CommandID{PluginID: one.GetPluginId(), Command: one.GetCommand()})
	}
	lists := h.listItems.Execute(ctx, h.vaults.GetCurrentVault(), r.Msg.GetPath(), commands)
	if err := ctx.Err(); err != nil {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}
	wire := make([]*v1.CommandItemList, 0, len(lists))
	for _, one := range lists {
		list := newWireItemList(one)
		list.PluginId, list.Command = one.PluginID, one.Command
		wire = append(wire, list)
	}
	return connect.NewResponse(&v1.ListCommandItemsResponse{Lists: wire}), nil
}

func newWireItemList(list plugincommand.ItemList) *v1.CommandItemList {
	if list.Err != nil {
		return &v1.CommandItemList{Error: getErrorCode(list.Err), Message: list.Err.Error()}
	}
	items := make([]*v1.CommandItem, 0, len(list.Items))
	for _, one := range list.Items {
		items = append(items, &v1.CommandItem{Id: one.ID, Title: one.Title, Detail: one.Detail})
	}
	return &v1.CommandItemList{Items: items}
}
