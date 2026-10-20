package source

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Transcript is the words made from a file, and whether anything was made.
type Transcript struct {
	Words  []byte
	Exists bool
}

// GetTranscript reads the text made from the file at a path, corrections first.
type GetTranscript struct {
	texts sourceTexts
}

func (u GetTranscript) Execute(ctx context.Context, v domain.Vault, path string) (Transcript, error) {
	stored, err := u.texts.getStoredText(ctx, v, path)
	if err != nil || !stored.isStored {
		return Transcript{}, err
	}
	producer, hash := stored.meta.Producer, stored.meta.Hash
	words, err := readFirst(ctx, stored.store,
		domain.NewArtifactName(domain.ArtifactCorrections, producer, hash),
		domain.NewArtifactName(domain.ArtifactText, producer, hash),
		domain.NewArtifactName(domain.ArtifactPartial, producer, hash),
	)
	return Transcript{Words: words, Exists: true}, err
}
