package service

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
)

// instructions is the paragraph that tells an agent when to call transcript_create.
const instructions = "Use transcript_create on an audio or video recording of the vault that has no transcript, " +
	"when its words are needed now. Transcription is slow, about as long as the recording itself. The call " +
	"returns a task id at once and the transcription runs in the background; the transcript can be " +
	"read and searched as it is written. Call it once per recording: a second call while the " +
	"first runs starts nothing and reports the state running, and a call on a recording already " +
	"transcribed starts nothing and reports the state done with the version library_read reads."

// The states transcript_create reports.
const (
	stateStarted = "started"
	stateRunning = "running"
	stateDone    = "done"
)

var (
	errNoVault     = errors.New("the call names no vault")
	errNotFound    = errors.New("the file is not found")
	errNoRecording = errors.New("the file is not an audio or video recording")
)

// TranscriptCreateInput is the input of the transcript_create tool.
type TranscriptCreateInput struct {
	Path string `json:"path" jsonschema:"the audio or video recording, relative to the vault root"`
}

// TranscriptCreateOutput is the output of the transcript_create tool.
type TranscriptCreateOutput struct {
	TaskID  string `json:"taskId" jsonschema:"the id of the transcription in the list of tasks"`
	Path    string `json:"path" jsonschema:"the recording; library_read and library_search read its transcript"`
	State   string `json:"state" jsonschema:"started; running where a transcription of this recording is already in progress and no second one was started; or done where the recording is already transcribed and nothing was started"`
	Version string `json:"version,omitempty" jsonschema:"with the state done, the id of the live text version, the version library_read reads: the transcript, or a version made from it such as a proofread"`
}

// Tools serves transcript_create. The transcriptions it starts end with Transcriber.Close.
type Tools struct {
	transcriber *Transcriber
	vault       host.Vault
}

// NewTools is the tools over the transcriber, checking the recordings named through the vault.
func NewTools(transcriber *Transcriber, vault host.Vault) *Tools {
	return &Tools{transcriber: transcriber, vault: vault}
}

// Registry registers and returns the transcription tool.
func (t *Tools) Registry() (*tools.Registry, error) {
	registry := tools.New()
	registry.SetInstructions(instructions)
	err := tools.Add(registry, tools.Spec{
		Step:  tools.ByEffect,
		Name:  "transcript_create",
		Title: "Transcribe a recording",
		Description: "Start the transcription of one audio or video recording of the vault, with the model the " +
			"plugin's setting selects, and return its task id without waiting. The transcript is stored as a " +
			"text version of the recording; library_search and library_read read it as it is written. A " +
			"transcription takes about as long as the recording. A file that is not in the vault, or is not an " +
			"audio or video recording, is refused. A recording that is already transcribed is not transcribed " +
			"again: the call answers the state done and the version library_read reads, the transcript or a " +
			"version made from it. A recording that cannot be read shows as the failure of its task in the list " +
			"of tasks.",
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
	if _, err := t.vault.With(vault).StatFile(ctx, in.Path); err != nil {
		if errors.Is(err, host.ErrNotExist) {
			return TranscriptCreateOutput{}, fmt.Errorf("%s: %w", in.Path, errNotFound)
		}
		return TranscriptCreateOutput{}, fmt.Errorf("stat %s: %w", in.Path, err)
	}
	if !isRecording(in.Path) {
		return TranscriptCreateOutput{}, fmt.Errorf("%s: %w", in.Path, errNoRecording)
	}
	out := TranscriptCreateOutput{TaskID: getTaskID(in.Path), Path: in.Path, State: stateRunning, Version: ""}
	if t.transcriber.isRunning(vault, in.Path) {
		return out, nil
	}
	version, isDone, err := t.transcriber.findLiveVersionFromWhole(ctx, vault, in.Path)
	if err != nil {
		return TranscriptCreateOutput{}, err
	}
	if isDone {
		out.State, out.Version = stateDone, string(version.GetID())
		return out, nil
	}
	isStarted, err := t.transcriber.Start(ctx, vault, in.Path)
	if err != nil {
		return TranscriptCreateOutput{}, err
	}
	if isStarted {
		out.State = stateStarted
	}
	return out, nil
}

// isRecording reports whether the extension of the file gives it an audio/* or video/* type, the
// types the manifest declares transcript.create for. The extensions are those shared-mime-info maps
// to the two types; the plugin cannot reach the host's table.
func isRecording(file string) bool {
	switch strings.ToLower(path.Ext(file)) {
	case ".aac", ".ac3", ".aif", ".aifc", ".aiff", ".amr", ".ape", ".au", ".flac", ".m4a", ".m4b",
		".mka", ".mp2", ".mp3", ".mpc", ".oga", ".ogg", ".opus", ".ra", ".snd", ".spx", ".wav", ".weba",
		".wma", ".wv":
		return true
	case ".3g2", ".3gp", ".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".mpeg", ".mpg", ".ogv", ".qt",
		".webm", ".wmv":
		return true
	}
	return false
}
