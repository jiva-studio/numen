package source

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/internal/urlfile"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"

	"github.com/jiva-studio/numen/modules/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/modules/libs/core/port/vaultport"
)

// sourceTexts finds what was made from a source and where it is kept.
type sourceTexts struct {
	readers vaultport.VaultReaders
	sources sourceport.SourceQueries
	derived vaultport.DerivedStores
}

// storedText is what was made from a source, the store holding it, and whether anything was.
type storedText struct {
	meta     sourceport.SourceText
	store    vaultport.DerivedStore
	isStored bool
}

func (s sourceTexts) isReady() bool {
	return s.readers != nil && s.sources != nil && s.derived != nil
}

// getStoredText is the text made from the file at a path, where there is one.
// A link note's text is named by its address; any other file's by the bytes the index recorded.
func (s sourceTexts) getStoredText(
	ctx context.Context, v domain.Vault, path string,
) (storedText, error) {
	if !s.isReady() {
		return storedText{}, nil
	}
	if pos := s.getNoteURL(ctx, v, path); string(pos) != "" {
		stored, _, err := s.getFetchedText(ctx, v, pos)
		return stored, err
	}
	return s.getIndexedText(ctx, v, path)
}

// getIndexedText is the text the index names for the file at a path.
func (s sourceTexts) getIndexedText(
	ctx context.Context, v domain.Vault, path string,
) (storedText, error) {
	meta, found, err := s.sources.GetSourceText(ctx, v.ID, path)
	if err != nil || !found || meta.Producer == "" {
		return storedText{}, err
	}
	store, err := s.derived.Open(v)
	if err != nil {
		return storedText{}, err
	}
	return storedText{meta: meta, store: store, isStored: true}, nil
}

// getNoteURL is the address the note at a path carries, and nothing where the file is not a note.
func (s sourceTexts) getNoteURL(ctx context.Context, v domain.Vault, path string) domain.URL {
	if s.readers == nil {
		return domain.URL("")
	}
	reader, err := s.readers.Open(v)
	if err != nil {
		return domain.URL("")
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return domain.URL("")
	}
	pos, err := urlfile.Read(raw)
	if err != nil {
		return domain.URL("")
	}
	return pos
}

// getFetchedText is the text kept under an address with how far its run has got, and a zero text where nothing was fetched.
func (s sourceTexts) getFetchedText(
	ctx context.Context, v domain.Vault, pos domain.URL,
) (storedText, RunProgress, error) {
	store, err := s.derived.Open(v)
	if err != nil {
		return storedText{}, RunProgress{}, err
	}
	hash := text.Fingerprint([]byte(string(pos)))
	for _, producer := range domain.GetAddressProducers() {
		progress, err := readRunProgress(ctx, store, producer, hash)
		if err != nil {
			return storedText{}, RunProgress{}, err
		}
		if progress.State != RunNone {
			return storedText{meta: sourceport.SourceText{Producer: producer, Hash: hash}, store: store, isStored: true}, progress, nil
		}
	}
	return storedText{}, RunProgress{}, nil
}
