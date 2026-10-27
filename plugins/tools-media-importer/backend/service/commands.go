package service

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// newCommands is the commands the manifest declares.
func newCommands(links *link.Links) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add("import", newImportCommand(links)); err != nil {
		return nil, err
	}
	return commands, nil
}

// newImportCommand makes the link note of an address in the root of the vault and fetches its text.
// A video published there is copied in the background.
func newImportCommand(links *link.Links) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		made, err := links.Create(ctx, "", req.Arguments.GetString("address"))
		if err != nil {
			return command.Result{}, err
		}
		if err := links.FetchText(ctx, made, true); err != nil {
			return command.Result{OpenedPath: made.Path}, fmt.Errorf("%s is made, and its text was not fetched: %w", made.Path, err)
		}
		message := fmt.Sprintf("%s is made and its text fetched", made.Path)
		if links.IsCopyable(made.Address) {
			links.StartCopy(made)
			message += "; the video is being copied"
		}
		return command.Result{Message: message, OpenedPath: made.Path}, nil
	}
}
