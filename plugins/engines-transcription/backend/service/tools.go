package service

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/api"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/transcribe"
)

// The instruction sentences of transcript_create: when to call it and what it returns.
const (
	transcriptUseSentence        = "Use transcript_create on an audio or video recording of the vault that has no transcript, when its words are needed now."
	transcriptSlowSentence       = "Transcription is slow, about as long as the recording itself."
	transcriptBackgroundSentence = "transcript_create returns a task id at once and the transcription runs in the background; the transcript can be read and searched as it is written."
	transcriptOnceSentence       = "Call transcript_create once per recording: a second call while the first runs starts nothing and reports the state running, and a call on a recording already transcribed starts nothing and reports the state done with the version that is read."
)

// toolset serves transcript_create. The transcriptions it starts end with the transcriber.
type toolset struct {
	requests *transcribe.Requests
}

func newTools(requests *transcribe.Requests) *toolset {
	return &toolset{requests: requests}
}

func (t *toolset) registry() (*tools.Registry, error) {
	registry := tools.New()
	err := tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "transcript_create",
		Title: "Transcribe a recording",
		Description: "Start the transcription of one audio or video recording of the vault, with the model the " +
			"plugin's setting selects, and return its task id without waiting. The transcript is stored as a " +
			"text version of the recording, which is read and searched as it is written. A " +
			"transcription takes about as long as the recording. A file that is not in the vault, or is not an " +
			"audio or video recording, is rejected. A recording that is already transcribed is not transcribed " +
			"again: the call returns the state done and the version that is read, the transcript or a " +
			"version made from it. A recording that cannot be read is reported as a failed task in the list " +
			"of tasks.",
		Effect:       tools.ReadWrite,
		Servers:      []tools.Server{tools.Full},
		Instructions: []string{transcriptUseSentence, transcriptSlowSentence, transcriptBackgroundSentence, transcriptOnceSentence},
	}, t.createTranscript)
	if err != nil {
		return nil, err
	}
	return registry, nil
}

func (t *toolset) createTranscript(ctx context.Context, in api.TranscriptCreateInput) (api.TranscriptCreateOutput, error) {
	outcome, err := t.requests.Create(ctx, domain.VaultID(tools.GetVault(ctx)), in.Path)
	if err != nil {
		return api.TranscriptCreateOutput{}, err
	}
	return api.TranscriptCreateOutput{TaskID: outcome.TaskID, Path: outcome.Path, State: outcome.State, Version: outcome.Version}, nil
}
