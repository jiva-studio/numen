package service

import (
	"context"
	"fmt"
	"path"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
)

// commandCreateText is the shared command the plugin handles by writing the text layer of a PDF.
const commandCreateText = "app:file.createText"

func newCommands(texts *TextServer) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add(commandCreateText, newCreateTextRun(texts)); err != nil {
		return nil, err
	}
	return commands, nil
}

func newCreateTextRun(texts *TextServer) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		if req.Path == "" {
			return command.Result{}, fmt.Errorf("%w: the text layer is extracted from a file", command.ErrBadArguments)
		}
		if err := texts.CreateText(ctx, TextRequest{Vault: "", Path: req.Path}); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: "Extracting the text layer of " + path.Base(req.Path), OpenedPath: ""}, nil
	}
}
