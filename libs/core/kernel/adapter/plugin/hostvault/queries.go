package hostvault

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"
)

type noteRefs interface {
	GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)
}

type noteTypes interface {
	GetTypes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteType, error)
}

type namedPaths interface {
	GetNamedPaths(ctx context.Context, vaultID domain.VaultID, name string) ([]string, error)
}

type typedPaths interface {
	GetPathsOfType(ctx context.Context, vaultID domain.VaultID, noteType domain.NoteType) ([]string, error)
}

type linkQueries interface {
	GetLinks(ctx context.Context, vaultID domain.VaultID, from string) ([]domain.ResolvedLink, error)
	GetBacklinks(ctx context.Context, vaultID domain.VaultID, to string) ([]domain.ResolvedLink, error)
	ResolveLinks(ctx context.Context, vaultID domain.VaultID, from string, targets []string) (map[string]domain.ResolvedLink, error)
}

// Lookups reads the index data of the notes and the links of the vault.
type Lookups struct {
	Notes      noteRefs
	Types      noteTypes
	NamedPaths namedPaths
	TypedPaths typedPaths
	Links      linkQueries

	session sessionAccess
}

// GetNotes returns the refs of the notes at the paths.
func (l Lookups) GetNotes(ctx context.Context, paths []string) (map[string]domain.NoteRef, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Notes.GetNotes(ctx, v.ID, paths)
}

// GetTypes returns the types of the notes at the paths.
func (l Lookups) GetTypes(ctx context.Context, paths []string) (map[string]domain.NoteType, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Types.GetTypes(ctx, v.ID, paths)
}

// GetNamedPaths returns the paths of the notes with the name.
func (l Lookups) GetNamedPaths(ctx context.Context, name string) ([]string, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.NamedPaths.GetNamedPaths(ctx, v.ID, name)
}

// GetPathsOfType returns the paths of the notes of the type.
func (l Lookups) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.TypedPaths.GetPathsOfType(ctx, v.ID, noteType)
}

// GetLinks returns the links of the note at the path.
func (l Lookups) GetLinks(ctx context.Context, from string) ([]domain.ResolvedLink, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Links.GetLinks(ctx, v.ID, from)
}

// GetBacklinks returns the links to the note at the path.
func (l Lookups) GetBacklinks(ctx context.Context, to string) ([]domain.ResolvedLink, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Links.GetBacklinks(ctx, v.ID, to)
}

// ResolveLinks resolves the addresses the note at the path links to.
func (l Lookups) ResolveLinks(
	ctx context.Context, from string, targets []string,
) (map[string]domain.ResolvedLink, error) {
	v, err := l.session.open()
	if err != nil {
		return nil, err
	}
	return l.Links.ResolveLinks(ctx, v.ID, from, targets)
}
