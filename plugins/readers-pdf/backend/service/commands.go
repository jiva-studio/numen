package service

import (
	"context"
	"fmt"
	"path"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
)

func newCommands(texts *extract.Writer) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add(command.CreateText, newCreateTextRun(texts)); err != nil {
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
