package artifacts

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetTranscriptCorrectionState says whether the corrections of a recording's transcript are RunComplete, RunActive or RunNone.
type GetTranscriptCorrectionState struct {
	Texts SourceTexts
}

func (u GetTranscriptCorrectionState) Execute(ctx context.Context, v domain.Vault, path string) (RunState, error) {
	stored, err := u.Texts.GetStoredText(ctx, v, path)
	if err != nil || !stored.isStored {
		return RunNone, err
	}
	meta := stored.meta
	switch _, err := stored.store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactCorrections, meta.Producer, meta.Hash)); {
	case err == nil:
		return RunComplete, nil
	case !errors.Is(err, fs.ErrNotExist):
		return RunNone, err
	}
	if !isUnclaimed(ctx, stored.store, domain.NewArtifactName(domain.ArtifactPartial, meta.Producer, meta.Hash)) {
		return RunActive, nil
	}
	return RunNone, nil
}
