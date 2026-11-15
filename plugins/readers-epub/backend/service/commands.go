package service

import (
	"context"
	"fmt"
	"path"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/usecase/texts"
)

func newCommands(writer *texts.Writer) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add(command.CreateText, newCreateTextRun(writer)); err != nil {
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
