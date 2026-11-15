package service

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/feed"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/usecase/transcribe"
)

// UpdateCommand is the command that transcribes the recordings of the shown vault that its file
// change feed names as added or changed. A trigger on file.added and file.changed runs it while
// Transcribe automatically is on; the recordings are read from the feed, not from the trigger.
const UpdateCommand = "transcripts.update"

func newCommands(transcriber *transcribe.Transcriber, requests *transcribe.Requests, follower *feed.Follower) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add(command.CreateText, newTranscribeCommand(transcriber)); err != nil {
		return nil, err
	}
	return commands, commands.Add(UpdateCommand, newUpdateCommand(requests, follower))
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

func newUpdateCommand(requests *transcribe.Requests, follower *feed.Follower) command.Run {
	return func(ctx context.Context, _ command.Request) (command.Result, error) {
		vault, err := requests.GetShownVaultID(ctx)
		if err != nil {
			return command.Result{}, err
		}
		if err := follower.Execute(ctx, vault); err != nil {
			return command.Result{}, err
		}
		return command.Result{}, nil
	}
}

func newFollower(requests *transcribe.Requests, feeds func(domain.VaultID) feed.Vault, report func(string, error)) *feed.Follower {
	return feed.New(feeds, requestRuns{requests: requests}, report)
}

// requestRuns starts the transcriptions the follower asks for through the requests.
type requestRuns struct {
	requests *transcribe.Requests
}

func (r requestRuns) Start(ctx context.Context, vault domain.VaultID, path string) (feed.State, error) {
	out, err := r.requests.Create(ctx, vault, path)
	if err != nil {
		return 0, err
	}
	switch out.State {
	case transcribe.StateDone:
		return feed.StateDone, nil
	case transcribe.StateRunning:
		return feed.StateRunning, nil
	}
	return feed.StateStarted, nil
}

func (r requestRuns) IsRunning(_ context.Context, vault domain.VaultID, path string) (bool, error) {
	return r.requests.IsRunning(vault, path), nil
}
