package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// trueValue is the text of a true boolean argument or setting.
const trueValue = "true"

// The shared commands the plugin handles.
const (
	commandImport        = command.ImportLink
	commandCreateText    = command.CreateText
	commandDownloadMedia = command.DownloadMedia
	commandDeleteMedia   = command.DeleteMedia
)

// argumentAddress is the argument of app:link.import that holds the web address.
const argumentAddress = "address"

func newCommands(links *link.Links) (*command.Registry, error) {
	commands := command.New()
	for id, run := range map[string]command.Run{
		commandImport:        newImportCommand(links),
		commandCreateText:    newDownloadTextCommand(links),
		commandDownloadMedia: newDownloadCopyCommand(links),
		commandDeleteMedia:   newDeleteCopyCommand(links),
	} {
		// gomutants:disable-next-line BRANCH_IF reason="registering distinct static command IDs on a new registry cannot fail"
		if err := commands.Add(id, run); err != nil {
			return nil, err
		}
	}
	return commands, nil
}

func newImportCommand(links *link.Links) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		made, err := links.Create(ctx, "", "", req.Arguments.GetString(argumentAddress))
		if err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: made.Path + " is made", OpenedPath: made.Path}, nil
	}
}

func newDownloadTextCommand(links *link.Links) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		found, err := links.Read(ctx, req.Path)
		if err != nil {
			return command.Result{}, err
		}
		if err := links.StartTextFetch(ctx, found); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: fmt.Sprintf("fetching the text of %s", req.Path), OpenedPath: ""}, nil
	}
}

func newDownloadCopyCommand(links *link.Links) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		found, err := links.Read(ctx, req.Path)
		if err != nil {
			return command.Result{}, err
		}
		size, isStarted, err := links.StartMediaCopy(ctx, found, download.MediaVideo)
		if errors.Is(err, link.ErrBeingFetched) {
			return command.Result{Message: fmt.Sprintf("the media of %s is already being downloaded", req.Path), OpenedPath: ""}, nil
		}
		if err != nil {
			return command.Result{}, err
		}
		if !isStarted {
			return command.Result{Message: fmt.Sprintf("%s of media are downloaded", describeSize(size)), OpenedPath: ""}, nil
		}
		return command.Result{Message: fmt.Sprintf("downloading the media of %s", req.Path), OpenedPath: ""}, nil
	}
}

func newDeleteCopyCommand(links *link.Links) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		found, err := links.Read(ctx, req.Path)
		if err != nil {
			return command.Result{}, err
		}
		if err := links.DeleteCopy(ctx, found); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: fmt.Sprintf("the media of %s are deleted", req.Path), OpenedPath: ""}, nil
	}
}

func describeSize(bytes int64) string {
	if bytes < 1<<20 {
		return fmt.Sprintf("%d bytes", bytes)
	}
	return fmt.Sprintf("%d MB", bytes>>20)
}
