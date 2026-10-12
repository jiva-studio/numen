package source

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
)

// GetTranscriptCorrectionState says whether the corrections of a recording's transcript are RunComplete, RunActive or RunNone.
type GetTranscriptCorrectionState struct {
	texts sourceTexts
}

func (u GetTranscriptCorrectionState) Execute(ctx context.Context, v domain.Vault, path string) (RunState, error) {
	stored, err := u.texts.getStoredText(ctx, v, path)
	if err != nil || !stored.isStored {
		return RunNone, err
	}
	meta := stored.meta
	switch _, err := stored.store.Read(ctx, text.Corrections(meta.Producer, meta.Hash)); {
	case err == nil:
		return RunComplete, nil
	case !errors.Is(err, fs.ErrNotExist):
		return RunNone, err
	}
	if !isUnclaimed(ctx, stored.store, text.Partial(meta.Producer, meta.Hash)) {
		return RunActive, nil
	}
	return RunNone, nil
}
