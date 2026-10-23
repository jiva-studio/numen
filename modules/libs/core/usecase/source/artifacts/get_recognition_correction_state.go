package artifacts

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetRecognitionCorrectionState says whether the corrections of a scan's recognition are RunComplete, RunActive or RunNone.
type GetRecognitionCorrectionState struct {
	Texts SourceTexts
}

func (u GetRecognitionCorrectionState) Execute(ctx context.Context, v domain.Vault, path string) (RunState, error) {
	stored, err := u.Texts.GetStoredText(ctx, v, path)
	if err != nil || !stored.isStored {
		return RunNone, err
	}
	hash := stored.meta.Hash
	switch _, err := stored.store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactProofread, domain.OCR, hash)); {
	case err == nil:
		return RunComplete, nil
	case !errors.Is(err, fs.ErrNotExist):
		return RunNone, err
	}
	switch _, err := stored.store.ReadArtifact(ctx, domain.NewArtifactName(domain.ArtifactCorrections, domain.OCR, hash)); {
	case err == nil:
		return RunActive, nil
	case !errors.Is(err, fs.ErrNotExist):
		return RunNone, err
	}
	return RunNone, nil
}
