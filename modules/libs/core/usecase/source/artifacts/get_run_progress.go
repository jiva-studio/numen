package artifacts

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// GetRunProgress says how far the run over the source at a path has got.
type GetRunProgress struct {
	Texts SourceTexts
}

func (u GetRunProgress) Execute(
	ctx context.Context, v domain.Vault, path string, kind domain.SourceKind,
) (RunProgress, error) {
	if !u.Texts.IsReady() {
		return RunProgress{}, nil
	}
	if pos := u.Texts.GetNoteURL(ctx, v, path); string(pos) != "" {
		_, progress, err := u.Texts.GetFetchedText(ctx, v, pos)
		return progress, err
	}
	stored, err := u.Texts.GetIndexedText(ctx, v, path)
	if err != nil {
		return RunProgress{}, err
	}
	if stored.isStored {
		return readRunProgress(ctx, stored.store, stored.meta.Producer, stored.meta.Hash)
	}
	// A recording the index names no producer for is read under the fingerprint of its bytes.
	if kind != domain.KindRecording {
		return RunProgress{}, nil
	}
	reader, err := u.Texts.Readers.Open(v)
	if err != nil {
		return RunProgress{}, err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return RunProgress{}, err
	}
	store, err := u.Texts.Derived.Open(v)
	if err != nil {
		return RunProgress{}, err
	}
	return readRunProgress(ctx, store, domain.ASR, text.Fingerprint(raw))
}
