// Package command answers a window's CommandService: the commands the enabled plugins declare, and
// running one of them.
package command

import (
	"context"
	"errors"
	"math"
	"net/http"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/plugincommand"
)

// VaultProvider names the vault the window shows. A window showing none answers a vault with no ID.
type VaultProvider interface {
	GetCurrentVault() domain.Vault
}

// Handler answers CommandService over the scenarios of the plugin commands, run over the shown vault.
type Handler struct {
	vaults     VaultProvider
	listing    plugincommand.ListCommands
	runCommand plugincommand.RunCommand
	listItems  plugincommand.ListItems
}

// NewHandler makes the handler over the scenarios that list and run the commands.
func NewHandler(
	vaults VaultProvider, listing plugincommand.ListCommands, runCommand plugincommand.RunCommand, listItems plugincommand.ListItems,
) *Handler {
	return &Handler{vaults: vaults, listing: listing, runCommand: runCommand, listItems: listItems}
}

// NewRoute is where CommandService is mounted, what the schema generated to answer it, and the
// handler it answers through; the window turns the three into a route.
func NewRoute(options connect.HandlerOption, handler *Handler) (path string, served http.Handler, service any) {
	path, served = numenv1connect.NewCommandServiceHandler(handler, options)
	return path, served, handler
}

// ListCommands lists the commands and the handlers of shared commands of every enabled plugin,
// whichever part of the plugin runs them; one with items, whose condition names version.*, is marked.
// It carries the content type of a file by its extension.
func (h *Handler) ListCommands(
	context.Context, *connect.Request[v1.ListCommandsRequest],
) (*connect.Response[v1.ListCommandsResponse], error) {
	listing := h.listing.Execute()
	commands := make([]*v1.PluginCommand, 0, len(listing.Commands))
	for _, one := range listing.Commands {
		commands = append(commands, newWireCommand(one))
	}
	slots := make([]*v1.PluginSlotEntry, 0, len(listing.Slots))
	for _, one := range listing.Slots {
		slots = append(slots, newWireSlotEntry(one))
	}
	handlers := make([]*v1.PluginHandler, 0, len(listing.Handlers))
	for _, one := range listing.Handlers {
		handlers = append(handlers, &v1.PluginHandler{
			PluginId: one.PluginID,
			Handler:  &v1.ManifestHandler{Command: one.Command.ID, Title: one.Command.Title, When: one.Command.When},
			HasItems: one.HasItems,
		})
	}
	return connect.NewResponse(&v1.ListCommandsResponse{
		Commands: commands, Slots: slots, ContentTypeByExtension: listing.ContentTypeByExtension, Handlers: handlers,
		Keybindings: newWireKeybindings(listing.Keybindings),
	}), nil
}

// RunPluginCommand runs one of them. A call that is not a command, matches none, fails, or is routed to no
// plugin is answered with an error code and a user-readable message.
func (h *Handler) RunPluginCommand(
	ctx context.Context, r *connect.Request[v1.RunPluginCommandRequest],
) (*connect.Response[v1.RunPluginCommandResponse], error) {
	result, err := h.runCommand.Execute(ctx, h.vaults.GetCurrentVault(), plugin.CommandCall{
		PluginID:  r.Msg.GetPluginId(),
		Command:   r.Msg.GetCommand(),
		Path:      r.Msg.GetPath(),
		Arguments: r.Msg.GetArguments(),
		ItemID:    r.Msg.GetItemId(),
	})
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, connect.NewError(connect.CodeCanceled, ctxErr)
		}
		return connect.NewResponse(&v1.RunPluginCommandResponse{Error: getErrorCode(err), Message: err.Error()}), nil
	}
	return connect.NewResponse(&v1.RunPluginCommandResponse{
		Message: result.Message, OpenedPath: result.OpenedPath,
	}), nil
}

func getErrorCode(err error) v1.CommandErrorCode {
	switch {
	case errors.Is(err, plugincommand.ErrUnknownCommand):
		return v1.CommandErrorCode_COMMAND_ERROR_CODE_UNKNOWN_COMMAND
	case errors.Is(err, plugincommand.ErrBadArguments):
		return v1.CommandErrorCode_COMMAND_ERROR_CODE_BAD_ARGUMENTS
	case errors.Is(err, domain.ErrUnavailable):
		return v1.CommandErrorCode_COMMAND_ERROR_CODE_UNAVAILABLE
	}
	return v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED
}

func newWireCommand(entry plugincommand.ListedCommand) *v1.PluginCommand {
	arguments := make([]*v1.CommandArgument, 0, len(entry.Command.Arguments))
	for _, one := range entry.Command.Arguments {
		arguments = append(arguments, &v1.CommandArgument{
			Name:       one.Name,
			Title:      one.Title,
			IsRequired: one.IsRequired,
		})
	}
	return &v1.PluginCommand{
		PluginId:  entry.PluginID,
		Id:        entry.Command.ID,
		Title:     entry.Command.Title,
		Icon:      entry.Command.Icon,
		Arguments: arguments,
		When:      entry.Command.When,
		HasItems:  entry.HasItems,
	}
}

func newWireKeybindings(entries []plugin.KeybindingEntry) []*v1.PluginKeybinding {
	wire := make([]*v1.PluginKeybinding, 0, len(entries))
	for _, one := range entries {
		wire = append(wire, &v1.PluginKeybinding{
			PluginId: one.PluginID,
			Keybinding: &v1.ManifestKeybinding{
				Key:     one.Keybinding.Key,
				Command: one.Keybinding.Command,
				Values:  one.Keybinding.Values,
				When:    one.Keybinding.When,
			},
		})
	}
	return wire
}

func newWireSlotEntry(one plugin.SlotEntry) *v1.PluginSlotEntry {
	entry := &v1.ManifestSlotEntry{
		Slot:   one.Entry.Slot,
		Title:  one.Entry.Title,
		Icon:   one.Entry.Icon,
		Group:  one.Entry.Group,
		When:   one.Entry.When,
		Values: one.Entry.Values,
	}
	switch {
	case one.Entry.Command != "":
		entry.Placed = &v1.ManifestSlotEntry_Command{Command: one.Entry.Command}
	case one.Entry.View != "":
		entry.Placed = &v1.ManifestSlotEntry_View{View: one.Entry.View}
	case one.Entry.Submenu != "":
		entry.Placed = &v1.ManifestSlotEntry_Submenu{Submenu: one.Entry.Submenu}
	}
	if one.Entry.Order != nil {
		order := clampToInt32(*one.Entry.Order)
		entry.Order = &order
	}
	return &v1.PluginSlotEntry{PluginId: one.PluginID, Entry: entry}
}

func clampToInt32(value int) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}
