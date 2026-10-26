package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// newCommands is the commands the manifest declares, each run to its end.
func newCommands(links *link.Links) (*command.Registry, error) {
	commands := command.New()
	for id, run := range map[string]command.Run{
		"import":        newImportCommand(links),
		"download-text": newDownloadTextCommand(links),
		"download-copy": newDownloadCopyCommand(links),
		"delete-copy":   newDeleteCopyCommand(links),
	} {
		if err := commands.Add(id, run); err != nil {
			return nil, err
		}
	}
	return commands, nil
}

// newImportCommand makes the link of an address in the root of the vault, fetches its text, and
// copies the video it points at.
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
			size, err := links.Copy(ctx, made)
			if err != nil {
				return command.Result{OpenedPath: made.Path}, fmt.Errorf("%s: %w", message, err)
			}
			message += fmt.Sprintf(", and a copy of %s kept", describeSize(size))
		}
		return command.Result{Message: message, OpenedPath: made.Path}, nil
	}
}

// newDownloadTextCommand fetches the text of a link again.
func newDownloadTextCommand(links *link.Links) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		found, err := links.Read(ctx, req.Path)
		if err != nil {
			return command.Result{}, err
		}
		if err := links.FetchText(ctx, found, true); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: fmt.Sprintf("the text of %s is fetched", req.Path)}, nil
	}
}

// newDownloadCopyCommand copies the video a link points at.
func newDownloadCopyCommand(links *link.Links) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		found, err := links.Read(ctx, req.Path)
		if err != nil {
			return command.Result{}, err
		}
		size, err := links.Copy(ctx, found)
		if errors.Is(err, link.ErrBeingFetched) {
			return command.Result{Message: fmt.Sprintf("a copy of %s is already being made", req.Path)}, nil
		}
		if err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: fmt.Sprintf("a copy of %s is kept", describeSize(size))}, nil
	}
}

// newDeleteCopyCommand takes away the copy kept of a link.
func newDeleteCopyCommand(links *link.Links) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		found, err := links.Read(ctx, req.Path)
		if err != nil {
			return command.Result{}, err
		}
		if err := links.DeleteCopy(ctx, found); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: fmt.Sprintf("the copy of %s is deleted", req.Path)}, nil
	}
}
