package command

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

// VersionItems lists the text versions of a file a command with items offers, and runs the command
// over one of them.
type VersionItems interface {
	ListVersionItems(ctx context.Context, v domain.Vault, path string, entry plugincommand.Entry) ([]textversion.VersionItem, error)
	RunVersionItem(ctx context.Context, v domain.Vault, path string, entry plugincommand.Entry, version domain.TextVersionID) (plugincommand.Result, error)
}

// errNoVersions is a command with items run where the window has no text versions.
var errNoVersions = errors.New("the window has no text versions")

// ListCommandItems lists the items each command of the request offers over the path, one list per
// command in the order of the request: the text versions of the file for a command with items, none
// for any other command. A command that is unknown, or whose items cannot be listed, has a list with
// the code and message of the error.
func (h *Handler) ListCommandItems(
	ctx context.Context, r *connect.Request[v1.ListCommandItemsRequest],
) (*connect.Response[v1.ListCommandItemsResponse], error) {
	lists := make([]*v1.CommandItemList, 0, len(r.Msg.GetCommands()))
	for _, one := range r.Msg.GetCommands() {
		list := h.listItems(ctx, r.Msg.GetPath(), one.GetPluginId(), one.GetCommand())
		list.PluginId, list.Command = one.GetPluginId(), one.GetCommand()
		lists = append(lists, list)
	}
	if err := ctx.Err(); err != nil {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}
	return connect.NewResponse(&v1.ListCommandItemsResponse{Lists: lists}), nil
}

func (h *Handler) listItems(ctx context.Context, path, pluginID, commandID string) *v1.CommandItemList {
	entry, found := h.findEntry(pluginID, commandID)
	if !found {
		return newItemsFailure(fmt.Errorf("%w: %s of %s", plugincommand.ErrUnknownCommand, commandID, pluginID))
	}
	if !plugincommand.HasItems(entry.Command) {
		return &v1.CommandItemList{}
	}
	if h.versions == nil {
		return newItemsFailure(errNoVersions)
	}
	versions, err := h.versions.ListVersionItems(ctx, h.vaults.GetCurrentVault(), path, entry)
	if err != nil {
		return newItemsFailure(err)
	}
	items := make([]*v1.CommandItem, 0, len(versions))
	for _, one := range versions {
		items = append(items, &v1.CommandItem{Id: string(one.Version), Title: getVersionTitle(one)})
	}
	return &v1.CommandItemList{Items: items}
}

func (h *Handler) runWithArgument(ctx context.Context, msg *v1.RunPluginCommandRequest) (plugincommand.Result, bool, error) {
	entry, found := h.findEntry(msg.GetPluginId(), msg.GetCommand())
	hasItems := found && plugincommand.HasItems(entry.Command)
	switch {
	case !hasItems && msg.GetArgument() == "":
		return plugincommand.Result{}, false, nil
	case !hasItems:
		return plugincommand.Result{}, true, fmt.Errorf("%w: %s offers no item", plugincommand.ErrBadArguments, msg.GetCommand())
	case msg.GetArgument() == "":
		return plugincommand.Result{}, true, fmt.Errorf("%w: %s runs over an item it offers", plugincommand.ErrBadArguments, msg.GetCommand())
	case h.versions == nil:
		return plugincommand.Result{}, true, errNoVersions
	}
	result, err := h.versions.RunVersionItem(ctx, h.vaults.GetCurrentVault(), msg.GetPath(), entry, domain.TextVersionID(msg.GetArgument()))
	if errors.Is(err, textversion.ErrInvalidVersion) {
		err = fmt.Errorf("%w: %w", plugincommand.ErrBadArguments, err)
	}
	return result, true, err
}

func (h *Handler) findEntry(pluginID, command string) (plugincommand.Entry, bool) {
	var entries []plugincommand.Entry
	if plugincommand.IsSharedCommand(command) {
		entries = h.list.ExecuteHandlers()
	} else {
		entries = h.list.ExecuteAll()
	}
	for _, one := range entries {
		if one.PluginID == pluginID && one.Command.ID == command {
			return one, true
		}
	}
	return plugincommand.Entry{}, false
}

func getVersionTitle(item textversion.VersionItem) string {
	kind := item.Kind
	if kind == "" {
		kind = "text"
	}
	if item.IsLive {
		return kind + " (live)"
	}
	return kind
}

func newItemsFailure(err error) *v1.CommandItemList {
	return &v1.CommandItemList{Error: getErrorCode(err), Message: err.Error()}
}
