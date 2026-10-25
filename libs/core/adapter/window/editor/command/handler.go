// Package command answers a window's CommandService: the commands the enabled plugins declare, and
// running one of them.
package command

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
)

// Handler answers CommandService over the scenarios of the plugin commands.
type Handler struct {
	list plugincommand.List
	run  plugincommand.Run
}

// NewHandler makes the handler over both scenarios.
func NewHandler(commands plugincommand.Commands) *Handler {
	return &Handler{list: commands.List, run: commands.Run}
}

// ListCommands lists the commands of every enabled plugin.
func (h *Handler) ListCommands(
	context.Context, *connect.Request[v1.ListCommandsRequest],
) (*connect.Response[v1.ListCommandsResponse], error) {
	listed := h.list.Execute()
	commands := make([]*v1.PluginCommand, 0, len(listed))
	for _, one := range listed {
		commands = append(commands, newWireCommand(one))
	}
	return connect.NewResponse(&v1.ListCommandsResponse{Commands: commands}), nil
}

// RunPluginCommand runs one of them. A call that is no command, does not meet one, fails, or reaches
// no plugin is answered with an error code and what a person reads about it.
func (h *Handler) RunPluginCommand(
	ctx context.Context, r *connect.Request[v1.RunPluginCommandRequest],
) (*connect.Response[v1.RunPluginCommandResponse], error) {
	result, err := h.run.Execute(ctx, plugincommand.Call{
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
		PluginId:       entry.PluginID,
		Id:             entry.Command.ID,
		Title:          entry.Command.Title,
		FileExtensions: entry.Command.FileExtensions,
		Arguments:      arguments,
	}
}
