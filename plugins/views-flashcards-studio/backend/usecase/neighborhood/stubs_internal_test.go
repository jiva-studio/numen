package neighborhood

import (
	"context"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

type stubVaultReaders struct {
	openErr error
	reader  ports.VaultReader
}

func (s stubVaultReaders) Open(_ domain.Vault) (ports.VaultReader, error) {
	if s.openErr != nil {
		return nil, s.openErr
	}
	return s.reader, nil
}

type stubVaultReader struct {
	statFn func(ctx context.Context, path string) (domain.FileStat, error)
	readFn func(ctx context.Context, path string) ([]byte, error)
}

func (s stubVaultReader) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	return s.statFn(ctx, path)
}

func (s stubVaultReader) Read(ctx context.Context, path string) ([]byte, error) {
	return s.readFn(ctx, path)
}

type stubLinkQueries struct {
	linksErr     error
	backlinksErr error
	links        map[string][]domain.ResolvedLink
	backlinks    map[string][]domain.ResolvedLink
}

func (s stubLinkQueries) GetLinks(_ context.Context, _ domain.VaultID, path string) ([]domain.ResolvedLink, error) {
	if s.linksErr != nil {
		return nil, s.linksErr
	}
	return s.links[path], nil
}

func (s stubLinkQueries) ResolveLinks(context.Context, domain.VaultID, string, []string) (map[string]domain.ResolvedLink, error) {
	return nil, nil
}

func (s stubLinkQueries) GetBacklinks(_ context.Context, _ domain.VaultID, path string) ([]domain.ResolvedLink, error) {
	if s.backlinksErr != nil {
		return nil, s.backlinksErr
	}
	return s.backlinks[path], nil
}

type stubNoteQueries struct {
	hasErr   error
	has      bool
	pathsErr map[domain.NoteType]error
	paths    map[domain.NoteType][]string
	notesErr error
	notes    map[string]domain.NoteRef
	typesErr error
	types    map[string]domain.NoteType
	namesErr error
	names    map[string][]string
}

func (s stubNoteQueries) Has(_ context.Context, _ domain.VaultID) (bool, error) {
	return s.has, s.hasErr
}

func (s stubNoteQueries) GetPathsOfType(
	_ context.Context, _ domain.VaultID, kind domain.NoteType,
) ([]string, error) {
	if err := s.pathsErr[kind]; err != nil {
		return nil, err
	}
	return s.paths[kind], nil
}

func (s stubNoteQueries) GetNotes(
	_ context.Context, _ domain.VaultID, _ []string,
) (map[string]domain.NoteRef, error) {
	if s.notesErr != nil {
		return nil, s.notesErr
	}
	return s.notes, nil
}

func (s stubNoteQueries) GetTypes(
	_ context.Context, _ domain.VaultID, _ []string,
) (map[string]domain.NoteType, error) {
	if s.typesErr != nil {
		return nil, s.typesErr
	}
	return s.types, nil
}

func (s stubNoteQueries) GetNamedPaths(
	_ context.Context, _ domain.VaultID, name string,
) ([]string, error) {
	if s.namesErr != nil {
		return nil, s.namesErr
	}
	return s.names[name], nil
}

func (s stubNoteQueries) GetCardModels(context.Context, domain.VaultID) ([]ports.CardModel, error) {
	return nil, nil
}
