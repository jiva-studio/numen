package transcribe

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/engines-transcription/backend/domain/recording"
)

// The states a request reports.
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

// Outcome is what a request to transcribe a recording reports. Version is set with the state done.
type Outcome struct {
	TaskID  string
	Path    string
	State   string
	Version string
}

// Requests starts the transcription of one recording of a vault, or reports the transcription that is
// already there.
type Requests struct {
	transcriber *Transcriber
	vaults      func(domain.VaultID) Vault
}

// NewRequests returns the requests over the transcriber, which check the requested recordings
// through the vault.
func NewRequests(transcriber *Transcriber, vaults func(domain.VaultID) Vault) *Requests {
	return &Requests{transcriber: transcriber, vaults: vaults}
}

// Create starts the transcription of the recording without waiting for it. A recording that is
// being transcribed starts nothing and reports the state running; a recording whose whole
// transcript is the live version, or the source of it, starts nothing and reports the state done.
func (r *Requests) Create(ctx context.Context, vault domain.VaultID, path string) (Outcome, error) {
	if vault == "" {
		return Outcome{}, errNoVault
	}
	if path == "" {
		return Outcome{}, errors.New("path required")
	}
	if _, err := r.vaults(vault).StatFile(ctx, path); err != nil {
		if errors.Is(err, host.ErrNotExist) {
			return Outcome{}, fmt.Errorf("%s: %w", path, errNotFound)
		}
		return Outcome{}, fmt.Errorf("stat %s: %w", path, err)
	}
	if !recording.IsRecording(path) {
		return Outcome{}, fmt.Errorf("%s: %w", path, errNoRecording)
	}
	out := Outcome{TaskID: getTaskID(vault, path), Path: path, State: stateRunning, Version: ""}
	if r.transcriber.isRunning(vault, path) {
		return out, nil
	}
	version, isDone, err := r.transcriber.findLiveVersionOfWholeTranscript(ctx, vault, path)
	if err != nil {
		return Outcome{}, err
	}
	if isDone {
		out.State, out.Version = stateDone, string(version.GetID())
		return out, nil
	}
	isStarted, err := r.transcriber.Start(ctx, vault, path)
	if err != nil {
		return Outcome{}, err
	}
	if isStarted {
		out.State = stateStarted
	}
	return out, nil
}
