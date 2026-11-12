package filetext

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/indexfake"
)

// bookReader is the plugin whose version every book the stand-in reads has.
const bookReader = "test.books"

// libraryTexts is the text versions of the libraries a test holds. A book the
// stand-in reads has a version of bookReader, made from the bytes it holds now;
// a test puts any other version with put.
type libraryTexts struct {
	shelves indexfake.Vaults

	mu      sync.Mutex
	extra   map[string][]placedVersion
	books   map[string]placedVersion
	failure error
	outside map[string]bool
}

type placedVersion struct {
	version domain.TextVersion
	parts   []domain.TextSegment
}

func newLibraryTexts(shelves indexfake.Vaults) *libraryTexts {
	return &libraryTexts{shelves: shelves, extra: map[string][]placedVersion{}, books: map[string]placedVersion{}, outside: map[string]bool{}}
}

func (s *libraryTexts) put(v domain.Vault, path, pluginID string, kind domain.TextKind, parts []domain.TextSegment) domain.TextVersion {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw := s.shelves[v.ID].Files[path].Raw
	version := domain.TextVersion{
		Path: path, PluginID: pluginID, Key: fmt.Sprint(len(s.extra[path])), Kind: kind, InputHash: domain.HashContent(raw),
	}
	s.extra[path] = append(s.extra[path], placedVersion{version: version, parts: parts})
	return version
}

func (s *libraryTexts) drop(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.extra, path)
}

func (s *libraryTexts) Open(v domain.Vault) (text.VaultTexts, error) {
	return libraryVaultTexts{texts: s, vault: v}, nil
}

type libraryVaultTexts struct {
	texts *libraryTexts
	vault domain.Vault
}

func (t libraryVaultTexts) findLiveVersion(ctx context.Context, path, contentHash string) (placedVersion, error) {
	s := t.texts
	s.mu.Lock()
	failure := s.failure
	extra := slices.Clone(s.extra[path])
	s.mu.Unlock()
	if failure != nil {
		return placedVersion{}, failure
	}
	for _, e := range slices.Backward(extra) {
		if e.version.InputHash == contentHash {
			return e, nil
		}
	}
	s.mu.Lock()
	made, isMade := s.books[path+"|"+contentHash]
	s.mu.Unlock()
	if isMade {
		return made, nil
	}
	file, ok := s.shelves[t.vault.ID].Files[path]
	if !ok || domain.HashContent(file.Raw) != contentHash {
		return placedVersion{}, text.ErrNoText
	}
	parts, err := indexfake.ReadBookSegments(file.Raw)
	if err != nil {
		return placedVersion{}, text.ErrNoText
	}
	version := domain.TextVersion{Path: path, PluginID: bookReader, Key: contentHash[:8], Kind: "book-text", InputHash: contentHash}
	made = placedVersion{version: version, parts: parts}
	s.mu.Lock()
	s.books[path+"|"+contentHash] = made
	s.mu.Unlock()
	return made, nil
}

func (t libraryVaultTexts) FindLiveVersion(ctx context.Context, path, contentHash string) (domain.TextVersionID, error) {
	live, err := t.findLiveVersion(ctx, path, contentHash)
	if err != nil {
		return "", err
	}
	return live.version.GetID(), nil
}

func (t libraryVaultTexts) GetLiveText(ctx context.Context, path, contentHash string) (domain.VersionText, error) {
	live, err := t.findLiveVersion(ctx, path, contentHash)
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
	file, ok := t.texts.shelves[t.vault.ID].Files[path]
	if !ok {
		return domain.VersionText{}, text.ErrNoText
	}
	live, err := t.findLiveVersion(ctx, path, domain.HashContent(file.Raw))
	if err != nil || live.version.GetID() != id {
		return domain.VersionText{}, text.ErrNoText
	}
	return domain.NewVersionText(live.version, live.parts), nil
}

// IsTextSource reports every file as a text source except those a test excluded with leaveOut.
func (t libraryVaultTexts) IsTextSource(_ context.Context, path string) (bool, error) {
	t.texts.mu.Lock()
	defer t.texts.mu.Unlock()
	if file, ok := t.texts.shelves[t.vault.ID].Files[path]; ok && file.Kind == domain.KindNote {
		return len(t.texts.extra[path]) > 0 && !t.texts.outside[path], nil
	}
	return !t.texts.outside[path], nil
}

func (s *libraryTexts) leaveOut(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outside[path] = true
}

// GetContentHash hashes the file from a stream of the fake vault, which counts it as a read.
func (t libraryVaultTexts) GetContentHash(ctx context.Context, path string) (_ string, err error) {
	file, err := t.texts.shelves[t.vault.ID].Open(ctx, path)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return domain.HashReader(file)
}
