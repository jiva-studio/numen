package source

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

type RunState int

const (
	// RunNone is a source no run has left anything of.
	RunNone RunState = iota
	// RunComplete is the whole text standing.
	RunComplete
	// RunActive is a run holding the source now.
	RunActive
	// RunStopped is part of the text on disk with no run behind it.
	RunStopped
	// RunNoSpeech is a run that found no speech.
	RunNoSpeech
	// RunUnopened is a run that could not open the bytes.
	RunUnopened
)

// RunProgress is how far a run has got: its state, why it got no words, and the bytes written so far.
type RunProgress struct {
	State RunState
	Why   string
	Size  int
}

// readRunProgress is how far the run keeping its files under a name has got.
func readRunProgress(ctx context.Context, store port.DerivedStore, producer, hash string) (RunProgress, error) {
	var progress RunProgress
	whole, err := store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactText, producer, hash))
	switch {
	case err == nil:
		progress.State, progress.Size = RunComplete, len(whole)
		return progress, nil
	case !errors.Is(err, fs.ErrNotExist):
		return progress, err
	}
	switch payload, err := store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactAnswer, producer, hash)); {
	case err == nil:
		switch answer, why := text.ReadAnswer(payload); answer {
		case text.Silent:
			progress.State = RunNoSpeech
		case text.Unopened:
			progress.State, progress.Why = RunUnopened, why
		}
		return progress, nil
	case !errors.Is(err, fs.ErrNotExist):
		return progress, err
	}

	// The partial file is read before the claim is tried, so the answer is the run's own file.
	part, err := store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactPartial, producer, hash))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return progress, err
	}
	progress.Size = len(part)
	switch {
	case !isUnclaimed(ctx, store, domain.NewArtifactName(domain.ArtifactPartial, producer, hash)):
		progress.State = RunActive
	case progress.Size > 0:
		progress.State = RunStopped
	}
	return progress, nil
}
