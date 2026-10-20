package source

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// TranscriptWrite is a transcript that was written; CutError is why it could not be cut again, and nil when it was.
type TranscriptWrite struct {
	CutError error
}

// WriteTranscript writes a person's words beside what the model heard, which stays under its own name.
type WriteTranscript struct {
	texts sourceTexts
}

// Execute writes the cues and then cuts the recording again with cut, which may be nil.
func (u WriteTranscript) Execute(
	ctx context.Context,
	v domain.Vault,
	path string,
	cues []domain.TranscriptCue,
	cut func(context.Context, domain.Vault, string) error,
) (TranscriptWrite, error) {
	stored, err := u.texts.getStoredText(ctx, v, path)
	if err != nil {
		return TranscriptWrite{}, err
	}
	if !stored.isStored {
		return TranscriptWrite{}, ErrNoTranscript
	}

	// A run holds the recording it is listening to by the name it appends to.
	release, err := stored.store.ClaimArtifact(ctx, domain.NewArtifactName(domain.ArtifactPartial, stored.meta.Producer, stored.meta.Hash))
	if errors.Is(err, port.ErrClaimed) {
		return TranscriptWrite{}, ErrTranscriptBusy
	}
	if err != nil {
		return TranscriptWrite{}, err
	}
	defer release()

	// The words are a person's, and a proofreader leaves them alone.
	written := append(transcript.Marshal(cues), transcript.GetHandWrittenMark()...)
	if err := stored.store.WriteArtifact(ctx, domain.NewArtifactName(domain.ArtifactCorrections, stored.meta.Producer, stored.meta.Hash), written); err != nil {
		return TranscriptWrite{}, err
	}
	if cut == nil {
		return TranscriptWrite{}, nil
	}
	return TranscriptWrite{CutError: cut(ctx, v, path)}, nil
}
