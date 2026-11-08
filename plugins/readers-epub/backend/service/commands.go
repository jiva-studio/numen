package service

import (
	"context"
	"fmt"
	"path"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
)

// commandCreateText is the shared command the plugin handles by writing the text of a book.
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
			return command.Result{}, fmt.Errorf("%w: the text is extracted from a book", command.ErrBadArguments)
		}
		if err := texts.CreateText(ctx, TextRequest{Vault: "", Path: req.Path}); err != nil {
			return command.Result{}, err
		}
		return command.Result{Message: "Extracting the text of " + path.Base(req.Path), OpenedPath: ""}, nil
	}
}
