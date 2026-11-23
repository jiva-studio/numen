package command

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// ListCommandItems lists the items each command of the request offers over the path, one list per
// command in the order of the request. A command that is unknown, or whose items cannot be listed, has a
// list with the code and message of the error; a command that names no plugin is unknown.
func (h *Handler) ListCommandItems(
	ctx context.Context, r *connect.Request[v1.ListCommandItemsRequest],
) (*connect.Response[v1.ListCommandItemsResponse], error) {
	commands := make([]plugincommand.CommandID, 0, len(r.Msg.GetCommands()))
	for _, one := range r.Msg.GetCommands() {
		commands = append(commands, plugincommand.CommandID{PluginID: one.GetPluginId(), Command: one.GetCommand()})
	}
	listed := h.listItems.Execute(ctx, h.vaults.GetCurrentVault(), r.Msg.GetPath(), withPlugin(commands))
	if err := ctx.Err(); err != nil {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}
	wire := make([]*v1.CommandItemList, 0, len(commands))
	next := 0
	for _, one := range commands {
		var list *v1.CommandItemList
		if one.PluginID == "" {
			list = newWireItemList(plugincommand.ItemList{Err: fmt.Errorf("%w: %w", plugin.ErrUnknownCommand, errNoPlugin)})
		} else {
			list = newWireItemList(listed[next])
			next++
		}
		list.PluginId, list.Command = one.PluginID, one.Command
		wire = append(wire, list)
	}
	return connect.NewResponse(&v1.ListCommandItemsResponse{Lists: wire}), nil
}

func withPlugin(commands []plugincommand.CommandID) []plugincommand.CommandID {
	named := make([]plugincommand.CommandID, 0, len(commands))
	for _, one := range commands {
		if one.PluginID != "" {
			named = append(named, one)
		}
	}
	return named
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
