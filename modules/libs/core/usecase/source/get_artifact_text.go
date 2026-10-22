package source

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// MostArtifactRead is how much of an artifact one read carries.
const MostArtifactRead = MostDocumentText

// ArtifactText is a range of an artifact's text. Start is what was asked for
// held within the text, and Length is how long the range is.
type ArtifactText struct {
	Artifact
	Words  string
	Start  int
	Length int
}

// GetArtifactText reads a range of one artifact as it relies on disk.
type GetArtifactText struct {
	shelf artifactShelf
}

func NewGetArtifactText(queries port.SourceQueries, derived port.DerivedStores) *GetArtifactText {
	return &GetArtifactText{shelf: artifactShelf{queries: queries, derived: derived}}
}

// Execute reads the range of the artifact of the given kind. A length of zero
// or less takes as much as one answer carries, and the range is cut on whole
// characters.
func (u *GetArtifactText) Execute(
	ctx context.Context, v domain.Vault, path, kind string, start, length int,
) (ArtifactText, error) {
	set, err := u.shelf.getArtifacts(ctx, v, path)
	if err != nil {
		return ArtifactText{}, err
	}
	for _, one := range set.Artifacts {
		if one.Kind != kind || !one.IsReadable() {
			continue
		}
		words, err := getArtifactWords(ctx, set.Store, one.Producer, set.Hash)
		if err != nil {
			return ArtifactText{}, err
		}
		at, run := getArtifactRange(words, start, length)
		return ArtifactText{Artifact: one, Words: run, Start: at, Length: len(run)}, nil
	}
	return ArtifactText{}, fmt.Errorf("nothing of the kind %q was made from %s", kind, path)
}

// getArtifactRange is the range of a text a call asked for, held within it and
// cut on whole characters.
func getArtifactRange(words string, start, length int) (int, string) {
	if length <= 0 || length > MostArtifactRead {
		length = MostArtifactRead
	}
	start = max(0, min(start, len(words)))
	length = min(length, len(words)-start)
	at, raw := text.WholeCharacters(start, []byte(words[start:start+length]), len(words))
	return at, string(raw)
}
