package service

import (
	"context"
	"fmt"
	"path"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/texts"
)

// UpdateCommand is the command that writes the texts of the EPUB books the file change feed of the
// shown vault names since the plugin last read it, and of the book it names first. A trigger on
// file.added runs it, before the host may have put the book in the feed.
const UpdateCommand = "bookTexts.update"

// ShownVault names the vault the host shows, which a command runs over.
type ShownVault func(ctx context.Context) (domain.Vault, error)

func newCommands(writer *texts.Writer, follower *texts.Follower, shown ShownVault) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add(command.CreateText, newCreateTextRun(writer)); err != nil {
		return nil, err
	}
	if err := commands.Add(UpdateCommand, newUpdateRun(follower, shown)); err != nil {
		return nil, err
	}
	return commands, nil
}

func newCreateTextRun(writer *texts.Writer) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		if req.Path == "" {
			return command.Result{}, fmt.Errorf("%w: the text is extracted from a book", command.ErrBadArguments)
		}
		if err := writer.CreateText(ctx, texts.Request{Vault: "", Path: req.Path}); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: "Extracting the text of " + path.Base(req.Path), OpenedPath: ""}, nil
	}
}

func newUpdateRun(follower *texts.Follower, shown ShownVault) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		v, err := shown(ctx)
		if err != nil {
			return command.Result{}, err
		}
		follower.Wake(ctx, v.ID, req.Path)
		return command.Result{}, nil
	}
}
