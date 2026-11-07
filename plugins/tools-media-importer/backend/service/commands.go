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
	commandImport        = "app:link.import"
	commandCreateText    = "app:file.createText"
	commandDownloadMedia = "app:link.downloadMedia"
	commandDeleteMedia   = "app:link.deleteMedia"
	commandDeleteText    = "app:version.delete"
)

// argumentAddress is the argument of app:link.import that holds the web address.
const argumentAddress = "address"

// newCommands is the handlers the manifest declares.
func newCommands(links *link.Links, hosted link.Host) (*command.Registry, error) {
	commands := command.New()
	for id, run := range map[string]command.Run{
		commandImport:        newImportCommand(links),
		commandCreateText:    newDownloadTextCommand(links),
		commandDownloadMedia: newDownloadCopyCommand(links),
		commandDeleteMedia:   newDeleteCopyCommand(links),
		commandDeleteText:    command.NewDeleteTextRun(hosted.Texts, hosted.Store, PluginID),
	} {
		// gomutants:disable-next-line BRANCH_IF reason="registering distinct static command IDs on a new registry cannot fail"
		if err := commands.Add(id, run); err != nil {
			return nil, err
		}
	}
	return commands, nil
}

// newImportCommand makes the link note of an address in the root of the vault, and nothing else.
func newImportCommand(links *link.Links) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		made, err := links.Create(ctx, "", "", req.Arguments.GetString(argumentAddress))
		if err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: made.Path + " is made", OpenedPath: made.Path}, nil
	}
}

// newDownloadTextCommand starts the fetch of the text of a link note and answers once its task is
// shown. The task shows how the fetch ends.
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

// newDownloadCopyCommand starts the download of the video a link note points at and answers once
// its task is shown. The task shows how the download ends.
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

// newDeleteCopyCommand deletes the media downloaded for a link note.
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

// describeSize returns a size in the unit a person reads it in.
func describeSize(bytes int64) string {
	if bytes < 1<<20 {
		return fmt.Sprintf("%d bytes", bytes)
	}
	return fmt.Sprintf("%d MB", bytes>>20)
}
