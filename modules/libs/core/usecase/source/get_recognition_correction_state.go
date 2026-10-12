package source

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
)

// GetRecognitionCorrectionState says whether the corrections of a scan's recognition are RunComplete, RunActive or RunNone.
type GetRecognitionCorrectionState struct {
	texts sourceTexts
}

func (u GetRecognitionCorrectionState) Execute(ctx context.Context, v domain.Vault, path string) (RunState, error) {
	stored, err := u.texts.getStoredText(ctx, v, path)
	if err != nil || !stored.isStored {
		return RunNone, err
	}
	hash := stored.meta.Hash
	switch _, err := stored.store.Read(ctx, text.Proofread(text.OCR, hash)); {
	case err == nil:
		return RunComplete, nil
	case !errors.Is(err, fs.ErrNotExist):
		return RunNone, err
	}
	switch _, err := stored.store.Read(ctx, text.Corrections(text.OCR, hash)); {
	case err == nil:
		return RunActive, nil
	case !errors.Is(err, fs.ErrNotExist):
		return RunNone, err
	}
	return RunNone, nil
}
