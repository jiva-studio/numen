package filetext

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
)

// bookReader is the plugin whose version every book the stand-in reads has.
const bookReader = "test.books"

// libraryTexts is the text versions of the libraries a test holds. A book the
// stand-in reads has a version of bookReader, made from the bytes it holds now;
// a test puts any other version with put.
type libraryTexts struct {
	shelves vaults

	mu    sync.Mutex
	extra map[string][]placedVersion
	// books is every version bookReader has made, by path and the hash of the bytes it read.
	books     map[string]placedVersion
	requested []string
	failure   error
	outside   map[string]bool
}

type placedVersion struct {
	version domain.TextVersion
	parts   []domain.TextSegment
}

func newLibraryTexts(shelves vaults) *libraryTexts {
	return &libraryTexts{shelves: shelves, extra: map[string][]placedVersion{}, books: map[string]placedVersion{}, outside: map[string]bool{}}
}

// put gives a file a version made from its bytes as they are now. The newest put is live.
func (s *libraryTexts) put(v domain.Vault, path, pluginID string, kind domain.TextKind, parts []domain.TextSegment) domain.TextVersion {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw := s.shelves[v.ID].files[path].raw
	version := domain.TextVersion{
		Path: path, PluginID: pluginID, Key: fmt.Sprint(len(s.extra[path])), Kind: kind, SourceHash: domain.HashContent(raw),
	}
	s.extra[path] = append(s.extra[path], placedVersion{version: version, parts: parts})
	return version
}

// drop removes every version a test put on a file.
func (s *libraryTexts) drop(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.extra, path)
}

func (s *libraryTexts) getRequested() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requested)
}

func (s *libraryTexts) Open(v domain.Vault) (textport.VaultTexts, error) {
	return libraryVaultTexts{texts: s, vault: v}, nil
}

type libraryVaultTexts struct {
	texts *libraryTexts
	vault domain.Vault
}

// findLive is the live version of a file and its parts.
func (t libraryVaultTexts) findLive(ctx context.Context, path, fileHash string) (placedVersion, error) {
	s := t.texts
	s.mu.Lock()
	failure := s.failure
	extra := slices.Clone(s.extra[path])
	s.mu.Unlock()
	if failure != nil {
		return placedVersion{}, failure
	}
	for i := len(extra) - 1; i >= 0; i-- {
		if extra[i].version.SourceHash == fileHash {
			return extra[i], nil
		}
	}
	s.mu.Lock()
	made, isMade := s.books[path+"|"+fileHash]
	s.mu.Unlock()
	if isMade {
		return made, nil
	}
	file, ok := s.shelves[t.vault.ID].files[path]
	if !ok || domain.HashContent(file.raw) != fileHash {
		return placedVersion{}, textport.ErrNoText
	}
	book, err := documents{}.ReadBook(ctx, file.raw)
	if err != nil {
		return placedVersion{}, textport.ErrNoText
	}
	version := domain.TextVersion{Path: path, PluginID: bookReader, Key: fileHash[:8], Kind: "book-text", SourceHash: fileHash}
	var parts []domain.TextSegment
	for i, part := range book.Parts {
		end := len(book.Text)
		if i+1 < len(book.Parts) {
			end = book.Parts[i+1].Offset
		}
		parts = append(parts, domain.TextSegment{Text: book.Text[part.Offset:end], Heading: part.Title})
	}
	made = placedVersion{version: version, parts: parts}
	s.mu.Lock()
	s.books[path+"|"+fileHash] = made
	s.mu.Unlock()
	return made, nil
}

func (t libraryVaultTexts) FindLiveVersion(ctx context.Context, path, fileHash string) (domain.TextVersionID, error) {
	live, err := t.findLive(ctx, path, fileHash)
	if err != nil {
		return "", err
	}
	return live.version.GetID(), nil
}

func (t libraryVaultTexts) GetLiveText(ctx context.Context, path, fileHash string) (domain.VersionText, error) {
	live, err := t.findLive(ctx, path, fileHash)
	if err != nil {
		return domain.VersionText{}, err
	}
	return domain.NewVersionText(live.version, live.parts), nil
}

func (t libraryVaultTexts) GetVersionText(ctx context.Context, path string, id domain.TextVersionID) (domain.VersionText, error) {
	t.texts.mu.Lock()
	extra := slices.Clone(t.texts.extra[path])
	t.texts.mu.Unlock()
	for _, one := range extra {
		if one.version.GetID() == id {
			return domain.NewVersionText(one.version, one.parts), nil
		}
	}
	file, ok := t.texts.shelves[t.vault.ID].files[path]
	if !ok {
		return domain.VersionText{}, textport.ErrNoText
	}
	live, err := t.findLive(ctx, path, domain.HashContent(file.raw))
	if err != nil || live.version.GetID() != id {
		return domain.VersionText{}, textport.ErrNoText
	}
	return domain.NewVersionText(live.version, live.parts), nil
}

func (t libraryVaultTexts) RequestText(_ context.Context, path, _ string) error {
	t.texts.mu.Lock()
	defer t.texts.mu.Unlock()
	t.texts.requested = append(t.texts.requested, path)
	return nil
}

// IsTextSource reports every file as a text source except those a test excluded with leaveOut.
func (t libraryVaultTexts) IsTextSource(_ context.Context, path string) (bool, error) {
	t.texts.mu.Lock()
	defer t.texts.mu.Unlock()
	return !t.texts.outside[path], nil
}

// leaveOut makes the file one that has no text versions and no plugin takes.
func (s *libraryTexts) leaveOut(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outside[path] = true
}
