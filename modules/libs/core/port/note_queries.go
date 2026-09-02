package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// NoteQueries answers questions about notes in shapes that are not notes.
type NoteQueries interface {
	// Fingerprints is what the index believes about each file, keyed by path, so
	// a scan can decide what to reparse without reading anything.
	Fingerprints(ctx context.Context, vaultID string) (map[string]domain.FileRef, error)

	// Search is the notes whose text matches the words typed. A search over
	// everything the vault holds is PassageQueries and the use case above it.
	Search(ctx context.Context, vaultID, query string, limit int) ([]domain.NoteMatch, error)

	// Names is the names in a vault that match the words typed: a note's own
	// title, and the headings inside notes. Searching the text a vault holds is
	// PassageQueries and the use case above it.
	Names(ctx context.Context, vaultID, query string, limit int) ([]domain.NameMatch, error)

	// Headings is what each of the notes asked about is divided into, in the
	// order they stand in it. A path that names nothing, and a note with no
	// headings, are absent from the answer.
	//
	// The byte a heading begins at is not in the index, so what comes back says
	// which line it stands on and nothing about where in the text that is.
	Headings(ctx context.Context, vaultID string, paths []string) (map[string][]domain.Heading, error)

	Summary(ctx context.Context, vaultID string) (domain.VaultSummary, error)

	// Notes returns what is needed to show a note, for the paths asked about.
	// Paths that name nothing are absent from the answer: a link resolves as
	// of now, and what it resolved to a moment ago may be gone.
	Notes(ctx context.Context, vaultID string, paths []string) (map[string]domain.NoteRef, error)

	// Opening is the note to show when nothing else has been chosen. False when
	// the vault holds none.
	Opening(ctx context.Context, vaultID string) (domain.NoteRef, bool, error)

	// Named is the paths of every note filed under one name. More than one is
	// what makes a link written by that name ambiguous.
	Named(ctx context.Context, vaultID, name string) ([]string, error)

	// Stencils is every stencil one vault holds, by path. A card names the
	// stencil it is cut by, and this is the list those names are picked from.
	Stencils(ctx context.Context, vaultID string) ([]Stencil, error)

	// Types is what each of the notes asked about is, keyed by path. A path
	// naming a file the index holds no note at is absent from the answer, and a
	// listing draws such an entry as the file it is.
	Types(ctx context.Context, vaultID string, paths []string) (map[string]domain.NoteType, error)

	// OfType is every note of one type the vault holds, by path. A caller after
	// the decks or the stencils of a vault asks for them, and opens no file to
	// find out what each note is.
	OfType(ctx context.Context, vaultID string, of domain.NoteType) ([]string, error)

	// Holds reports whether the index carries this vault at all. A vault it
	// does not carry is one nothing has scanned yet, and a caller that only
	// reads the index tells a person so rather than showing them a vault that
	// looks empty.
	Holds(ctx context.Context, vaultID string) (bool, error)
}

// Stencil is one stencil as a caller choosing between them sees it: where the
// file is, and what it is called.
type Stencil struct {
	Path  string
	Title string
}

// LinkQueries answers what points where. It is separate from NoteQueries
// because a use case about links has no business being handed a search.
type LinkQueries interface {
	// Links returns what one note points at, resolved as of now.
	Links(ctx context.Context, vaultID, from string) ([]domain.ResolvedLink, error)
	// Backlinks returns what points at one note, by whichever address form was
	// written: its identifier, or a name that resolves to it.
	Backlinks(ctx context.Context, vaultID, to string) ([]domain.ResolvedLink, error)

	// Resolve returns where each of those addresses lands, written in one note
	// and keyed by what was written. It answers the question a link is answered
	// with, asked about text nobody recorded as a link: the wikilink a card
	// names its stencil by. An address that reaches nothing is absent.
	Resolve(ctx context.Context, vaultID, from string, written []string) (map[string]domain.ResolvedLink, error)
}
