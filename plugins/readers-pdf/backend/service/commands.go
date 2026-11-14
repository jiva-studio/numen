package service

import (
	"context"
	"fmt"
	"path"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
)

type commands struct {
	texts *extract.Writer
}

func newCommands(texts *extract.Writer) command.Handler {
	return commands{texts: texts}
}

func (c commands) RunCommand(ctx context.Context, req command.Request) (command.Result, error) {
	if req.Command != command.CreateText {
		return command.Result{}, fmt.Errorf("%w: %s", command.ErrUnknownCommand, req.Command)
	}
	if req.Path == "" {
		return command.Result{}, fmt.Errorf("%w: the text layer is extracted from a file", command.ErrBadArguments)
	}
	if err := c.texts.CreateText(ctx, extract.Request{Vault: "", Path: req.Path}); err != nil {
		return command.Result{}, err
	}
	return command.Result{Message: "Extracting the text layer of " + path.Base(req.Path), OpenedPath: ""}, nil
}
