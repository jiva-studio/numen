package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
)

// VersionItems lists the text versions of a file a command with items offers, and runs the command
// over one of them; it lists every version of a file and deletes one for Delete text.
type VersionItems interface {
	ListVersionItems(ctx context.Context, v domain.Vault, path string, entry plugin.CommandEntry) ([]textversion.VersionItem, error)
	RunVersionItem(ctx context.Context, v domain.Vault, path string, entry plugin.CommandEntry, version domain.TextVersionID) (plugin.CommandResult, error)
	ListEveryVersion(ctx context.Context, v domain.Vault, path string) ([]textversion.VersionItem, error)
	DeleteVersion(ctx context.Context, v domain.Vault, path string, version domain.TextVersionID) error
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
	if isDeleteText(pluginID, commandID) {
		return h.listEveryVersion(ctx, path)
	}
	entry, found := h.findEntry(pluginID, commandID)
	if !found {
		return newItemsFailure(fmt.Errorf("%w: %s of %s", plugincommand.ErrUnknownCommand, commandID, pluginID))
	}
	if !h.list.HasItems(entry.Command) {
		return &v1.CommandItemList{}
	}
	if h.versions == nil {
		return newItemsFailure(errNoVersions)
	}
	versions, err := h.versions.ListVersionItems(ctx, h.vaults.GetCurrentVault(), path, entry)
	if err != nil {
		return newItemsFailure(err)
	}
	return h.newVersionList(versions)
}

func (h *Handler) listEveryVersion(ctx context.Context, path string) *v1.CommandItemList {
	if h.versions == nil {
		return newItemsFailure(errNoVersions)
	}
	versions, err := h.versions.ListEveryVersion(ctx, h.vaults.GetCurrentVault(), path)
	if err != nil {
		return newItemsFailure(err)
	}
	return h.newVersionList(versions)
}

func (h *Handler) newVersionList(versions []textversion.VersionItem) *v1.CommandItemList {
	items := make([]*v1.CommandItem, 0, len(versions))
	for _, one := range versions {
		parts := h.list.GetParts(one.PluginID)
		items = append(items, &v1.CommandItem{
			Id: string(one.Version), Title: getVersionTitle(one, parts), Detail: getVersionDetail(one, parts),
		})
	}
	return &v1.CommandItemList{Items: items}
}

func isDeleteText(pluginID, command string) bool {
	return pluginID == "" && command == textversion.DeleteCommand
}

func (h *Handler) deleteVersion(ctx context.Context, msg *v1.RunPluginCommandRequest) error {
	switch {
	case msg.GetItemId() == "":
		return fmt.Errorf("%w: %s deletes the version its item names", plugincommand.ErrBadArguments, msg.GetCommand())
	case h.versions == nil:
		return errNoVersions
	}
	err := h.versions.DeleteVersion(ctx, h.vaults.GetCurrentVault(), msg.GetPath(), domain.TextVersionID(msg.GetItemId()))
	if errors.Is(err, textversion.ErrInvalidVersion) {
		return fmt.Errorf("%w: %w", plugincommand.ErrBadArguments, err)
	}
	return err
}

func (h *Handler) runWithArgument(ctx context.Context, msg *v1.RunPluginCommandRequest) (plugin.CommandResult, bool, error) {
	if isDeleteText(msg.GetPluginId(), msg.GetCommand()) {
		return plugin.CommandResult{}, true, h.deleteVersion(ctx, msg)
	}
	entry, found := h.findEntry(msg.GetPluginId(), msg.GetCommand())
	hasItems := found && h.list.HasItems(entry.Command)
	switch {
	case !hasItems:
		return plugin.CommandResult{}, false, nil
	case msg.GetItemId() == "":
		return plugin.CommandResult{}, true, fmt.Errorf("%w: %s runs over an item it offers", plugincommand.ErrBadArguments, msg.GetCommand())
	case h.versions == nil:
		return plugin.CommandResult{}, true, errNoVersions
	}
	result, err := h.versions.RunVersionItem(ctx, h.vaults.GetCurrentVault(), msg.GetPath(), entry, domain.TextVersionID(msg.GetItemId()))
	if errors.Is(err, textversion.ErrInvalidVersion) {
		err = fmt.Errorf("%w: %w", plugincommand.ErrBadArguments, err)
	}
	return result, true, err
}

func (h *Handler) findEntry(pluginID, command string) (plugin.CommandEntry, bool) {
	var entries []plugin.CommandEntry
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
	return plugin.CommandEntry{}, false
}

func getVersionTitle(item textversion.VersionItem, parts plugin.Parts) string {
	title := item.Kind
	if declared, ok := parts.TextKinds[item.Kind]; ok {
		title = declared
	}
	if title == "" {
		title = "text"
	}
	if item.IsLive {
		return title + " (current)"
	}
	return title
}

func getVersionDetail(item textversion.VersionItem, parts plugin.Parts) string {
	maker := parts.Name
	if maker == "" {
		maker = item.PluginID
	}
	if item.CreateTime.IsZero() {
		return maker
	}
	if maker == "" {
		return item.CreateTime.Format(time.DateOnly)
	}
	return maker + " · " + item.CreateTime.Format(time.DateOnly)
}

func newItemsFailure(err error) *v1.CommandItemList {
	return &v1.CommandItemList{Error: getErrorCode(err), Message: err.Error()}
}
