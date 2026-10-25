package artifacts

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/internal/text"
	"github.com/jiva-studio/numen/libs/core/internal/urlfile"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/sourceport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// SourceTexts finds what was made from a source and where it is kept.
type SourceTexts struct {
	Readers vaultport.VaultReaders
	Sources sourceport.SourceQueries
	Derived vaultport.DerivedStores

	readers vaultport.VaultReaders
	sources sourceport.SourceQueries
	derived vaultport.DerivedStores
}

// StoredText is what was made from a source, the store holding it, and whether anything was.
type StoredText struct {
	Meta     sourceport.SourceText
	Store    vaultport.DerivedStore
	IsStored bool

	meta     sourceport.SourceText
	store    vaultport.DerivedStore
	isStored bool
}

func NewSourceTexts(readers vaultport.VaultReaders, sources sourceport.SourceQueries, derived vaultport.DerivedStores) SourceTexts {
	return SourceTexts{
		Readers: readers, Sources: sources, Derived: derived,
		readers: readers, sources: sources, derived: derived,
	}
}

func (s SourceTexts) IsReady() bool {
	return s.Readers != nil && s.Sources != nil && s.Derived != nil
}

// GetStoredText is the text made from the file at a path, where there is one.
// A link note's text is named by its address; any other file's by the bytes the index recorded.
func (s SourceTexts) GetStoredText(
	ctx context.Context, v domain.Vault, path string,
) (StoredText, error) {
	if !s.IsReady() {
		return StoredText{}, nil
	}
	if pos := s.GetNoteURL(ctx, v, path); string(pos) != "" {
		stored, _, err := s.GetFetchedText(ctx, v, pos)
		return stored, err
	}
	return s.GetIndexedText(ctx, v, path)
}

// GetIndexedText is the text the index names for the file at a path.
func (s SourceTexts) GetIndexedText(
	ctx context.Context, v domain.Vault, path string,
) (StoredText, error) {
	meta, found, err := s.sources.GetSourceText(ctx, v.ID, path)
	if err != nil || !found || meta.Producer == "" {
		return StoredText{}, err
	}
	store, err := s.derived.Open(v)
	if err != nil {
		return StoredText{}, err
	}
	return StoredText{
		Meta: meta, Store: store, IsStored: true,
		meta: meta, store: store, isStored: true,
	}, nil
}

// GetNoteURL is the address the note at a path carries, and nothing where the file is not a note.
func (s SourceTexts) GetNoteURL(ctx context.Context, v domain.Vault, path string) domain.URL {
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

// GetFetchedText is the text kept under an address with how far its run has got, and a zero text where nothing was fetched.
func (s SourceTexts) GetFetchedText(
	ctx context.Context, v domain.Vault, pos domain.URL,
) (StoredText, RunProgress, error) {
	store, err := s.derived.Open(v)
	if err != nil {
		return StoredText{}, RunProgress{}, err
	}
	hash := text.Fingerprint([]byte(string(pos)))
	for _, producer := range domain.GetAddressProducers() {
		progress, err := readRunProgress(ctx, store, producer, hash)
		if err != nil {
			return StoredText{}, RunProgress{}, err
		}
		if progress.State != RunNone {
			return StoredText{
				Meta: sourceport.SourceText{Producer: producer, Hash: hash}, Store: store, IsStored: true,
				meta: sourceport.SourceText{Producer: producer, Hash: hash}, store: store, isStored: true,
			}, progress, nil
		}
	}
	return StoredText{}, RunProgress{}, nil
}
