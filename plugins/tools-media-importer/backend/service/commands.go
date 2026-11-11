package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

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
		found, isFound, err := links.FindOrCreate(ctx, "", "", req.Arguments.GetString(argumentAddress))
		if err != nil {
			return command.Result{}, err
		}
		if isFound {
			return command.Result{Message: "Opened " + found.Path, OpenedPath: found.Path}, nil
		}
		return command.Result{Message: "Created " + found.Path, OpenedPath: found.Path}, nil
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
		return command.Result{Message: "Fetching the text of " + req.Path, OpenedPath: ""}, nil
	}
}

func newDownloadCopyCommand(links *link.Links) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		media, err := getMedia(req.ItemID)
		if err != nil {
			return command.Result{}, err
		}
		found, err := links.Read(ctx, req.Path)
		if err != nil {
			return command.Result{}, err
		}
		size, isStarted, err := links.StartMediaCopy(ctx, found, media)
		if errors.Is(err, link.ErrBeingFetched) {
			return command.Result{Message: "Already downloading the media of " + req.Path, OpenedPath: ""}, nil
		}
		if err != nil {
			return command.Result{}, err
		}
		if !isStarted {
			return command.Result{Message: fmt.Sprintf("Downloaded the media of %s (%s)", req.Path, link.GetSizeText(size)), OpenedPath: ""}, nil
		}
		return command.Result{Message: "Downloading the media of " + req.Path, OpenedPath: ""}, nil
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
		return command.Result{Message: "Deleted the media of " + req.Path, OpenedPath: ""}, nil
	}
}

func getMedia(item string) (download.Media, error) {
	// A run with no item, as numen-cli makes, downloads the video.
	media, isKnown := download.ParseMedia(item)
	if !isKnown {
		return "", fmt.Errorf("%w: %s offers no item %q", command.ErrBadArguments, commandDownloadMedia, item)
	}
	return media, nil
}
