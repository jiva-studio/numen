// Package command answers a window's CommandService: the commands the enabled plugins declare, and
// running one of them.
package command

import (
	"context"
	"errors"
	"math"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// VaultProvider names the vault the window shows. A window showing none answers a vault with no ID.
type VaultProvider interface {
	GetCurrentVault() domain.Vault
}

// Handler answers CommandService over the scenarios of the plugin commands, run over the shown vault.
type Handler struct {
	vaults VaultProvider
	list   plugincommand.List
	run    plugincommand.Run
}

// NewHandler makes the handler over both scenarios.
func NewHandler(vaults VaultProvider, commands plugincommand.Commands) *Handler {
	return &Handler{vaults: vaults, list: commands.List, run: commands.Run}
}

// ListCommands lists the commands of every enabled plugin, whichever part of the plugin runs them,
// except a command with items: its condition names version.*, the menu of a file offers it over a
// version, and run without one it fails. It carries the content type of a file by its extension.
func (h *Handler) ListCommands(
	context.Context, *connect.Request[v1.ListCommandsRequest],
) (*connect.Response[v1.ListCommandsResponse], error) {
	entries := h.list.ExecuteAll()
	commands := make([]*v1.PluginCommand, 0, len(entries))
	for _, one := range entries {
		if plugincommand.HasItems(one.Command) {
			continue
		}
		commands = append(commands, newWireCommand(one))
	}
	slotEntries := h.list.ExecuteSlots()
	slots := make([]*v1.PluginSlotEntry, 0, len(slotEntries))
	for _, one := range slotEntries {
		slots = append(slots, newWireSlotEntry(one))
	}
	return connect.NewResponse(&v1.ListCommandsResponse{
		Commands: commands, Slots: slots, ContentTypeByExtension: h.list.ListContentTypes(),
	}), nil
}

// RunPluginCommand runs one of them. A call that is not a command, matches none, fails, or is routed to
// no plugin is answered with an error code and a user-readable message.
func (h *Handler) RunPluginCommand(
	ctx context.Context, r *connect.Request[v1.RunPluginCommandRequest],
) (*connect.Response[v1.RunPluginCommandResponse], error) {
	result, err := h.run.Execute(ctx, h.vaults.GetCurrentVault(), plugincommand.Call{
		PluginID:  r.Msg.GetPluginId(),
		Command:   r.Msg.GetCommand(),
		Path:      r.Msg.GetPath(),
		Arguments: r.Msg.GetArguments(),
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
	case errors.Is(err, plugincommand.ErrUnavailable):
		return v1.CommandErrorCode_COMMAND_ERROR_CODE_UNAVAILABLE
	}
	return v1.CommandErrorCode_COMMAND_ERROR_CODE_FAILED
}

// newWireCommand is a command as the schema carries it.
func newWireCommand(entry plugincommand.Entry) *v1.PluginCommand {
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
		Shortcut:  entry.Command.Shortcut,
	}
}

// newWireSlotEntry is a slot entry as the schema carries it.
func newWireSlotEntry(one plugincommand.SlotEntry) *v1.PluginSlotEntry {
	entry := &v1.ManifestSlotEntry{
		Slot:  one.Entry.Slot,
		Title: one.Entry.Title,
		Icon:  one.Entry.Icon,
		Group: one.Entry.Group,
		When:  one.Entry.When,
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

// clampToInt32 narrows an order to the 32 bits the schema carries.
func clampToInt32(value int) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}
