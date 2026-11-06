package service

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// Instructions is the paragraph that tells an agent when to call transcript_create.
const Instructions = "Use transcript_create on an audio or video recording of the vault that has no transcript, " +
	"when its words are needed now. Transcription is slow, about as long as the recording itself. The call " +
	"returns a task id at once and the transcription runs in the background; library_search and " +
	"library_read read the transcript as it is written. Call it once per recording: a second call while the " +
	"first runs starts nothing and reports the state running."

// The states transcript_create reports.
const (
	stateStarted = "started"
	stateRunning = "running"
)

var errNoVault = errors.New("the call names no vault")

// TranscriptCreateInput is the input of the transcript_create tool.
type TranscriptCreateInput struct {
	Path string `json:"path" jsonschema:"the audio or video recording, relative to the vault root"`
}

// TranscriptCreateOutput is the output of the transcript_create tool.
type TranscriptCreateOutput struct {
	TaskID string `json:"task_id" jsonschema:"the id of the transcription in the list of tasks"`
	Path   string `json:"path" jsonschema:"the recording; library_read and library_search read its transcript"`
	State  string `json:"state" jsonschema:"started, or running where a transcription of this recording is already in progress and no second one was started"`
}

// Tools serves transcript_create. The transcriptions it starts end with Transcriber.Close.
type Tools struct {
	transcriber *Transcriber
}

// NewTools is the tools over the transcriber.
func NewTools(transcriber *Transcriber) *Tools {
	return &Tools{transcriber: transcriber}
}

// Registry registers and returns the transcription tool.
func (t *Tools) Registry() (*tools.Registry, error) {
	registry := tools.New()
	err := tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "transcript_create",
		Title: "Transcribe a recording",
		Description: "Start the transcription of one audio or video recording of the vault, with the model the " +
			"plugin's setting selects, and return its task id without waiting. The transcript is stored as a " +
			"text version of the recording; library_search and library_read read it as it is written. A " +
			"transcription takes about as long as the recording. A recording that is not in the vault is refused.",
		Effect:  tools.ReadWrite,
		Servers: []tools.Server{tools.Full},
	}, t.createTranscript)
	if err != nil {
		return nil, err
	}
	return registry, nil
}

func (t *Tools) createTranscript(ctx context.Context, in TranscriptCreateInput) (TranscriptCreateOutput, error) {
	vault := domain.VaultID(tools.GetVault(ctx))
	if vault == "" {
		return TranscriptCreateOutput{}, errNoVault
	}
	if in.Path == "" {
		return TranscriptCreateOutput{}, errors.New("path required")
	}
	isStarted, err := t.transcriber.Start(ctx, vault, in.Path)
	if err != nil {
		return TranscriptCreateOutput{}, err
	}
	state := stateRunning
	if isStarted {
		state = stateStarted
	}
	return TranscriptCreateOutput{TaskID: getTaskID(in.Path), Path: in.Path, State: state}, nil
}
