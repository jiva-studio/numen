// Package textmenu answers a window's TextVersionService: what the menu of a file
// of the shown vault offers from its text versions, and running an entry.
package textmenu

import (
	"context"
	"errors"
	"io/fs"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

// Menu is the menu of a file as its text versions make it.
type Menu interface {
	GetMenu(ctx context.Context, v domain.Vault, path string) ([]textversion.MenuEntry, error)
	RunMenuEntry(ctx context.Context, v domain.Vault, path string, entry textversion.MenuEntry) error
}

// VaultProvider names the vault the window shows. A window showing none answers a vault with no ID.
type VaultProvider interface {
	GetCurrentVault() domain.Vault
}

// Handler answers TextVersionService over the shown vault. A handler with no menu offers nothing.
type Handler struct {
	vaults VaultProvider
	menu   Menu
}

// NewHandler is the handler over the menu of the shown vault's files.
func NewHandler(vaults VaultProvider, menu Menu) *Handler {
	return &Handler{vaults: vaults, menu: menu}
}

// errNoVault is a call while the window shows no vault.
var errNoVault = errors.New("the window shows no vault")

var actions = map[textversion.MenuAction]v1.FileMenuAction{
	textversion.ActionCreateText:    v1.FileMenuAction_FILE_MENU_ACTION_CREATE_TEXT,
	textversion.ActionDeleteVersion: v1.FileMenuAction_FILE_MENU_ACTION_DELETE_VERSION,
}

// GetFileMenu lists the entries of the file's menu, for the file as it is now.
func (h *Handler) GetFileMenu(
	ctx context.Context, r *connect.Request[v1.GetFileMenuRequest],
) (*connect.Response[v1.GetFileMenuResponse], error) {
	v := h.vaults.GetCurrentVault()
	if h.menu == nil || v.ID == "" {
		return connect.NewResponse(&v1.GetFileMenuResponse{}), nil
	}
	entries, err := h.menu.GetMenu(ctx, v, r.Msg.GetPath())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.FileMenuEntry, 0, len(entries))
	for _, one := range entries {
		out = append(out, &v1.FileMenuEntry{
			Action: actions[one.Action], PluginId: one.PluginID, PluginName: one.PluginName, VersionId: string(one.Version),
			IsLive: one.IsLive, Title: one.Item.Title, Icon: one.Item.Icon,
		})
	}
	return connect.NewResponse(&v1.GetFileMenuResponse{Entries: out}), nil
}

// RunFileMenuEntry does what the entry offers. An entry that names no action, or a plugin or a
// version the file's menu does not hold, is an invalid argument.
func (h *Handler) RunFileMenuEntry(
	ctx context.Context, r *connect.Request[v1.RunFileMenuEntryRequest],
) (*connect.Response[v1.RunFileMenuEntryResponse], error) {
	v := h.vaults.GetCurrentVault()
	if h.menu == nil || v.ID == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errNoVault)
	}
	entry := textversion.MenuEntry{PluginID: r.Msg.GetPluginId(), Version: domain.TextVersionID(r.Msg.GetVersionId())}
	for action, wire := range actions {
		if wire == r.Msg.GetAction() {
			entry.Action = action
		}
	}
	err := h.menu.RunMenuEntry(ctx, v, r.Msg.GetPath(), entry)
	switch {
	case err == nil:
		return connect.NewResponse(&v1.RunFileMenuEntryResponse{}), nil
	case errors.Is(err, textversion.ErrInvalidVersion), errors.Is(err, textversion.ErrSourceNotAccepted),
		errors.Is(err, textversion.ErrNotTextPlugin), errors.Is(err, textversion.ErrNoText):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return nil, connect.NewError(connect.CodeInternal, err)
}
