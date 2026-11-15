package service

import (
	"context"
	"fmt"
	"path"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
)

// UpdateCommand is the command that writes the text layers of the PDF files the file change feed of
// the shown vault names since the plugin last read it, and of the file it names first. A trigger on
// file.added runs it, before the host may have put the file in the feed.
const UpdateCommand = "textLayers.update"

// ShownVault names the vault the host shows, which a command runs over.
type ShownVault interface {
	GetVault(ctx context.Context) (domain.Vault, error)
}

func newCommands(texts *extract.Writer, follower *extract.Follower, shown ShownVault) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add(command.CreateText, newCreateTextRun(texts)); err != nil {
		return nil, err
	}
	if err := commands.Add(UpdateCommand, newUpdateRun(follower, shown)); err != nil {
		return nil, err
	}
	return commands, nil
}

func newCreateTextRun(texts *extract.Writer) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		if req.Path == "" {
			return command.Result{}, fmt.Errorf("%w: the text layer is extracted from a file", command.ErrBadArguments)
		}
		if err := texts.CreateText(ctx, extract.Request{Vault: "", Path: req.Path}); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: "Extracting the text layer of " + path.Base(req.Path), OpenedPath: ""}, nil
	}
}

func newUpdateRun(follower *extract.Follower, shown ShownVault) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		v, err := shown.GetVault(ctx)
		if err != nil {
			return command.Result{}, err
		}
		follower.Wake(ctx, v.ID, req.Path)
		return command.Result{}, nil
	}
}
