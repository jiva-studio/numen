package service

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/transcribe"
)

func newCommands(transcriber *transcribe.Transcriber) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add(command.CreateText, newTranscribeCommand(transcriber)); err != nil {
		return nil, err
	}
	return commands, nil
}

func newTranscribeCommand(transcriber *transcribe.Transcriber) command.Run {
	return func(ctx context.Context, req command.Request) (command.Result, error) {
		isStarted, err := transcriber.Start(ctx, "", req.Path)
		if err != nil {
			return command.Result{}, err
		}
		if !isStarted {
			return command.Result{}, fmt.Errorf("%s: %w", req.Path, transcribe.ErrAlreadyTranscribing)
		}
		return command.Result{Message: "transcribing " + req.Path, OpenedPath: ""}, nil
	}
}
