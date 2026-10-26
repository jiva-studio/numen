package noteport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// NoteQueries answers questions about notes in shapes that are not notes.
type NoteQueries interface {
	// GetFingerprints is what the index believes about each file, keyed by path, so
	// a scan can decide what to reparse without reading anything.
	GetFingerprints(ctx context.Context, vaultID domain.VaultID) (map[string]domain.Fingerprint, error)

	// GetNameMatches is the names in a vault that match the words typed: a note's own
	// title, and the headings inside notes. Searching the text a vault holds is
	// PassageQueries and the use case above it.
	GetNameMatches(ctx context.Context, vaultID domain.VaultID, query string, limit int) ([]domain.NameMatch, error)

	// GetHeadings is what each of the notes asked about is divided into, in the
	// order they stand in it. A path that is empty, and a note with no
	// headings, are absent from the answer.
	//
	// The byte a heading begins at is not in the index, so what comes back says
	// which line it relies on and nothing about where in the text that is.
	GetHeadings(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string][]domain.Heading, error)

	GetSummary(ctx context.Context, vaultID domain.VaultID) (domain.VaultSummary, error)

	// GetNotes returns what is needed to show a note, for the paths asked about.
	// Paths that name nothing are absent from the answer: a link resolves as
	// of now, and what it resolved to a moment ago may be gone.
	GetNotes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteRef, error)

	// GetOpeningNote is the note to show when nothing else has been chosen. False when
	// the vault holds none.
	GetOpeningNote(ctx context.Context, vaultID domain.VaultID) (domain.NoteRef, bool, error)

	// GetNamedPaths is the paths of every note filed under one name. More than
	// one is what makes a link written by that name ambiguous.
	GetNamedPaths(ctx context.Context, vaultID domain.VaultID, name string) ([]string, error)

	// GetTypes is what each of the notes asked about is, keyed by path. A path
	// naming a file the index holds no note at is absent from the answer, and a
	// listing draws such an entry as the file it is.
	GetTypes(ctx context.Context, vaultID domain.VaultID, paths []string) (map[string]domain.NoteType, error)

	// GetPathsOfType is every note of one type the vault holds, by path. A caller after
	// the notes of one type asks for them, and opens no file to
	// find out what each note is.
	GetPathsOfType(ctx context.Context, vaultID domain.VaultID, noteType domain.NoteType) ([]string, error)

	// Has reports whether the index carries this vault at all. A vault it does
	// not carry is one nothing has scanned yet, and a caller that only reads
	// the index tells a person so.
	Has(ctx context.Context, vaultID domain.VaultID) (bool, error)
}

// LinkQueries answers what points where. It is separate from NoteQueries
// because a use case about links has no business being handed a search.
type LinkQueries interface {
	// GetLinks returns what one note points at, resolved as of now.
	GetLinks(ctx context.Context, vaultID domain.VaultID, from string) ([]domain.ResolvedLink, error)
	// GetBacklinks returns what points at one note, by whichever address form was
	// written: its identifier, or a name that resolves to it.
	GetBacklinks(ctx context.Context, vaultID domain.VaultID, to string) ([]domain.ResolvedLink, error)

	// ResolveLinks returns where each of those addresses lands, written in one note
	// and keyed by what was written. It answers the question a link is answered
	// with, asked about text nobody recorded as a link, such as a wikilink a
	// plugin reads out of a note. An address that reaches nothing is absent.
	ResolveLinks(ctx context.Context, vaultID domain.VaultID, from string, content []string) (map[string]domain.ResolvedLink, error)
}
