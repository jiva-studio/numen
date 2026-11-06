package command

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
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

// ListCommandItems lists the items the command offers over the path: the text versions of the file
// for a command with items, none for any other command.
func (h *Handler) ListCommandItems(
	ctx context.Context, r *connect.Request[v1.ListCommandItemsRequest],
) (*connect.Response[v1.ListCommandItemsResponse], error) {
	entry, found := h.findEntry(r.Msg.GetPluginId(), r.Msg.GetCommand())
	if !found {
		return newItemsFailure(fmt.Errorf("%w: %s of %s", plugincommand.ErrUnknownCommand, r.Msg.GetCommand(), r.Msg.GetPluginId())), nil
	}
	if !plugincommand.HasItems(entry.Command) {
		return connect.NewResponse(&v1.ListCommandItemsResponse{}), nil
	}
	if h.versions == nil {
		return newItemsFailure(errNoVersions), nil
	}
	versions, err := h.versions.ListVersionItems(ctx, h.vaults.GetCurrentVault(), r.Msg.GetPath(), entry)
	if err != nil {
		return newItemsFailure(err), nil
	}
	items := make([]*v1.CommandItem, 0, len(versions))
	for _, one := range versions {
		items = append(items, &v1.CommandItem{Id: string(one.Version), Title: getVersionTitle(one)})
	}
	return connect.NewResponse(&v1.ListCommandItemsResponse{Items: items}), nil
}

// runWithArgument runs the command of a call that names an item, or a command with items, over the
// version the argument names. It reports false for any other call, which runs as a plain command.
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

// findEntry is the command of the id an enabled plugin declares, whichever part of it runs the command,
// or its handler of the shared command of the id.
func (h *Handler) findEntry(pluginID, command string) (plugincommand.Entry, bool) {
	entries := h.list.ExecuteAll()
	if pluginport.IsSharedCommand(command) {
		entries = h.list.ExecuteHandlers()
	}
	for _, one := range entries {
		if one.PluginID == pluginID && one.Command.ID == command {
			return one, true
		}
	}
	return plugincommand.Entry{}, false
}

// getVersionTitle is what a person reads a version as: its kind, and whether it is the live one.
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

// newItemsFailure answers a list of items that could not be made with the code and message of the error.
func newItemsFailure(err error) *connect.Response[v1.ListCommandItemsResponse] {
	return connect.NewResponse(&v1.ListCommandItemsResponse{Error: getErrorCode(err), Message: err.Error()})
}
