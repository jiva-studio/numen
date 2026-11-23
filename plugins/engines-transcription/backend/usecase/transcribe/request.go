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
	StateStarted = "started"
	StateRunning = "running"
	StateDone    = "done"
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
	getVault    func(domain.VaultID) Vault
}

// NewRequests returns the requests over the transcriber, which check the requested recordings
// through the vault.
func NewRequests(transcriber *Transcriber, vaults func(domain.VaultID) Vault) *Requests {
	return &Requests{transcriber: transcriber, getVault: vaults}
}

// IsRunning reports whether a transcription of the recording of the vault runs.
func (r *Requests) IsRunning(vault domain.VaultID, path string) bool {
	return r.transcriber.isRunning(vault, path)
}

// GetShownVaultID asks the host which vault its window has open.
func (r *Requests) GetShownVaultID(ctx context.Context) (domain.VaultID, error) {
	return r.transcriber.getClaimVault(ctx, "")
}

// Create starts the transcription of the recording without waiting for it. A recording that is
// being transcribed starts nothing and reports the state running; a recording whose whole
// transcript is the live version, or the source of it, starts nothing and reports the state done.
func (r *Requests) Create(ctx context.Context, vault domain.VaultID, path string) (Outcome, error) {
	return r.create(ctx, vault, path, r.transcriber.Start)
}

// CreateInBackground starts the transcription as Create does, as work the plugin started on its own,
// which waits behind the work people asked for.
func (r *Requests) CreateInBackground(ctx context.Context, vault domain.VaultID, path string) (Outcome, error) {
	return r.create(ctx, vault, path, r.transcriber.StartInBackground)
}

func (r *Requests) create(
	ctx context.Context, vault domain.VaultID, path string, start func(context.Context, domain.VaultID, string) (bool, error),
) (Outcome, error) {
	if vault == "" {
		return Outcome{}, errNoVault
	}
	if path == "" {
		return Outcome{}, errors.New("path required")
	}
	if _, err := r.getVault(vault).StatFile(ctx, path); err != nil {
		if errors.Is(err, host.ErrNotExist) {
			return Outcome{}, fmt.Errorf("%s: %w", path, errNotFound)
		}
		return Outcome{}, fmt.Errorf("stat %s: %w", path, err)
	}
	if !recording.IsRecording(path) {
		return Outcome{}, fmt.Errorf("%s: %w", path, errNoRecording)
	}
	out := Outcome{TaskID: getTaskID(vault, path), Path: path, State: StateRunning, Version: ""}
	if r.transcriber.isRunning(vault, path) {
		return out, nil
	}
	version, isDone, err := r.transcriber.findLiveVersionOfWholeTranscript(ctx, vault, path)
	if err != nil {
		return Outcome{}, err
	}
	if isDone {
		out.State, out.Version = StateDone, string(version.GetID())
		return out, nil
	}
	isStarted, err := start(ctx, vault, path)
	if err != nil {
		return Outcome{}, err
	}
	if isStarted {
		out.State = StateStarted
	}
	return out, nil
}
