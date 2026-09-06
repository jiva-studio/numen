// Package note stores notes and answers questions about them. The repository
// and the queries are separate types on purpose: a repository is a collection of
// notes, and a search result or a count is not a note.
package note

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/modules/libs/core/adapter/index/writing"
	"github.com/jiva-studio/numen/modules/libs/core/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/format"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

//go:embed sql/*.sql
var files embed.FS

var stmt = sqlfile.Load(files, "sql")

// kind is what a note is among the sources the index holds. Everything this
// package writes and reads is one, and a path naming a file of another kind
// answers nothing here.
const kind = "note"

// Repository is the collection of notes. It puts one in and takes one out, and
// answers no questions about them.
type Repository struct {
	db *sql.DB

	// sizes are what a note is cut at, and reads what a chunk of it has to read
	// like to be kept. A repository told neither takes what the chunking
	// package names.
	sizes chunking.Sizes
	reads chunking.Legibility
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// Cut is the repository, cutting a note at the sizes given and keeping what
// reads as text. The settings decide both, and what has read the settings
// passes them in here.
func (r *Repository) Cut(sizes chunking.Sizes, reads chunking.Legibility) *Repository {
	return &Repository{db: r.db, sizes: sizes, reads: reads}
}

// exec runs a named statement and says which one failed. A bare driver error
// from one of the many statements in a transaction is a schema mistake nobody
// can locate.
func exec(ctx context.Context, tx *writing.Transaction, name string, args ...any) error {
	if _, err := tx.ExecContext(ctx, stmt.Get(name), args...); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// Save writes a group of notes in one transaction.
//
// A note and the size and date that call it up to date are stored together or
// not at all, so an interrupted scan leaves files to be read again.
func (r *Repository) Save(
	ctx context.Context, vaultID domain.VaultID, notes []domain.IndexedNote,
) error {
	if len(notes) == 0 {
		return nil
	}
	tx, err := writing.Begin(ctx, r.db)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	vault, err := vaultRow(ctx, tx, vaultID)
	if err != nil {
		return err
	}
	for _, one := range notes {
		if err := saveNote(ctx, tx, vault, one, r.sizes, r.reads); err != nil {
			return fmt.Errorf("%s: %w", one.Note.Fingerprint.Path, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func saveNote(
	ctx context.Context, tx *writing.Transaction, vault int64, indexed domain.IndexedNote,
	sizes chunking.Sizes, reads chunking.Legibility,
) error {
	n, artifact := indexed.Note, indexed.Artifact
	// Where a link note points is stored so that a sweep can ask what still
	// names an artifact. Every other note points nowhere.
	var address string
	if n.Type == domain.TypeLink {
		if at, wrong := domain.ReadAddress(n.Frontmatter); len(wrong) == 0 {
			address = at.URL
		}
	}
	frontmatter, storeErr := encodeFrontmatter(n)
	problem := n.FrontmatterErr
	if storeErr != "" {
		// The parser promises that a file which cannot be understood is still
		// indexed. Frontmatter that cannot be stored is the same case: the note
		// goes in without it, and the reason is recorded beside the note rather
		// than ending the scan of the whole vault.
		problem = strings.TrimSpace(problem + "\n" + storeErr)
	}

	var row int64
	if err := tx.QueryRowContext(ctx, stmt.Get("save_source"),
		vault, n.Fingerprint.Path, kind, n.Fingerprint.Size, chunk.Stamp(n.Fingerprint.ModTime),
		nullable(addressHash(address, artifact)), nullable(recipeOf(artifact, sizes)), nullable(artifact.Producer),
	).Scan(&row); err != nil {
		return fmt.Errorf("record the source this note is: %w", err)
	}
	if err := exec(ctx, tx, "save_note", row, vault, domain.FoldName(domain.Basename(n.Fingerprint.Path)),
		n.Title, string(noteType(n)), nullable(n.ID), frontmatter, nullable(problem),
		nullable(address)); err != nil {
		return err
	}

	// The source row survives a re-save, so nothing cascades and each kind of
	// derived row is cleared by hand.
	for _, name := range []string{
		"clear_heading_names", "clear_headings", "clear_links", "clear_problems",
	} {
		if err := exec(ctx, tx, name, row); err != nil {
			return err
		}
	}
	kept := outline(n)

	// The note goes in as its own large chunk, so the words in it are findable
	// as soon as it is indexed. A chunk whose text is what it was keeps its
	// row, and the vector made from it.
	if err := chunk.Replace(ctx, tx, row, vault, cut(n, artifact, kept, sizes, reads)); err != nil {
		return err
	}
	for _, h := range kept {
		if err := exec(ctx, tx, "insert_heading", row, h.Line, h.Level, h.Text); err != nil {
			return err
		}
	}

	// The names this note answers to, indexed for the words in them: its own
	// title, and every heading inside it. The headings go in once they are all
	// stored, because the index of them is keyed by the numbers they were
	// stored under.
	if err := exec(ctx, tx, "insert_title", row, n.Title); err != nil {
		return err
	}
	if err := exec(ctx, tx, "insert_heading_names", row); err != nil {
		return err
	}
	for i, l := range n.Links {
		if _, err := tx.ExecContext(ctx, stmt.Get("insert_link"), row, i,
			l.Target.Scheme, l.Target.Value, domain.FoldName(domain.LinkName(l.Target.Value)),
			string(l.Role), nullable(l.Type), nullable(l.Why), nullable(l.Label),
		); err != nil {
			return fmt.Errorf("store what this note points at: %w", err)
		}
	}
	for _, detail := range n.Problems {
		if err := exec(ctx, tx, "insert_problem", row, detail); err != nil {
			return err
		}
	}
	return nil
}

// cut is how a note is cut: one large chunk over the whole of it, and the small
// chunks inside that carry the vectors. The small chunks are cut at the sizes a
// book's are, and a mixed vault ranks by what a passage says.
//
// The large chunk is the note itself. The sizes decide the small chunks inside
// it.
//
// The note's headings are its parts, so the small chunks of one section are
// tiled inside that section and no chunk runs across a heading.
//
// Offsets are into the file. The body begins after the frontmatter, and every
// chunk is moved out by as much. A link note is the exception: its offsets are
// into its prose and what was fetched taken together, which is the text
// whatever reads a passage back out of one composes.
//
// A deck and a stencil are cut into nothing. A card is found by its heading,
// which is its question, and a stencil by its title, which is its file name.
// The vectors hang off the chunks, so neither is embedded either.
func cut(
	n domain.Note, artifact domain.Artifact, headings []domain.Heading,
	sizes chunking.Sizes, reads chunking.Legibility,
) []chunk.Chunk {
	if n.Type == domain.TypeDeck || n.Type == domain.TypeStencil {
		return nil
	}

	at := int(n.Fingerprint.Size) - len(n.Body)
	if at < 0 {
		at = 0
	}
	sizes.Large = chunking.Whole
	body, divisions := n.Body, parts(headings)

	// A link note is cut over what its person wrote and what was fetched from
	// the address, as one text: an offset is into the two together, and the
	// join is a part so that no chunk runs out of one into the other.
	if !artifact.IsZero() {
		at = 0
		divisions = append(divisions, chunking.PartStart{Offset: len(body) + len(text.Separator)})
		body += text.Separator + artifact.Text
	}

	out := make([]chunk.Chunk, 0, 1)
	for _, large := range chunking.Cut(body, divisions, sizes, reads) {
		// The title is searched together with the body: a note is looked for by
		// the name it was given.
		c := chunk.Chunk{
			Start:    at + large.Start,
			Length:   large.Length,
			Location: large.Location,
			Text:     n.Title + "\n" + large.Slice(body),
		}
		for _, small := range large.Small {
			c.Small = append(c.Small, chunk.Chunk{
				Start:    at + small.Start,
				Length:   small.Length,
				Location: small.Location,
				Text:     small.Slice(body),
			})
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		// A note of a title and no words is answered by its title.
		out = append(out, chunk.Chunk{Start: at, Length: len(body), Text: n.Title})
	}
	return out
}

// address is the hash a link note's text is kept under, which is the address it
// points at and not the bytes of the file. A person typing in the note changes
// the file and not what is at the address, and what was fetched is theirs to
// keep. Every other note is made from itself and names nothing.
func addressHash(address string, artifact domain.Artifact) string {
	if artifact.IsZero() {
		return ""
	}
	return text.Fingerprint([]byte(address))
}

// recipeOf names what produced a link note's text and the sizes it was cut into.
// A note carrying another one owes its text again.
func recipeOf(artifact domain.Artifact, sizes chunking.Sizes) string {
	if artifact.IsZero() {
		return ""
	}
	return fmt.Sprintf("%s+%s/large=%d+%d/small=%d+%d/limit=%d",
		text.ReaderNote, artifact.Producer,
		sizes.Large, sizes.LargeOverlap, sizes.Small, sizes.SmallOverlap, sizes.Limit)
}

// parts is where a note names the section that follows. A heading carries the
// byte its line begins at in the body, which is the offset a chunk is cut
// against, and the heading's own text is what the section is called.
//
// They are the headings the index keeps, so a passage is announced under a name
// somebody wrote.
func parts(headings []domain.Heading) []chunking.PartStart {
	if len(headings) == 0 {
		return nil
	}
	out := make([]chunking.PartStart, 0, len(headings))
	for _, h := range headings {
		out = append(out, chunking.PartStart{Title: h.Text, Offset: h.Offset})
	}
	return out
}

// The two levels a deck spends on what a person writes: a section, and a card
// under it. Below them stand the stencil's field names, written out under every
// card.
//
// TODO: which level a deck spends on what is the format package's answer, and
// it is where the format is read. Take these from there once it names them.
const (
	sectionLevel = 1
	cardLevel    = 2
)

// outline is the headings of a note as the index keeps them.
//
// A deck keeps its sections and its cards, and a card's heading is kept without
// the mark it ends in. A card whose first field is empty is named by nothing,
// and a heading of no text is not kept. A stencil keeps none: its headings are
// its faces and their two sides. Every other note keeps every heading it has.
func outline(n domain.Note) []domain.Heading {
	switch n.Type {
	case domain.TypeStencil:
		return nil
	case domain.TypeDeck:
		out := make([]domain.Heading, 0, len(n.Headings))
		for _, h := range n.Headings {
			switch h.Level {
			case sectionLevel:
				out = append(out, h)
			case cardLevel:
				h.Text, _ = format.ReadHeading(h.Text)
				if h.Text == "" {
					continue
				}
				out = append(out, h)
			}
		}
		return out
	default:
		return n.Headings
	}
}

func (r *Repository) Remove(ctx context.Context, vaultID domain.VaultID, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	tx, err := writing.Begin(ctx, r.db)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	vault, err := vaultRow(ctx, tx, vaultID)
	if err != nil {
		return err
	}
	for _, path := range paths {
		var row int64
		err := tx.QueryRowContext(ctx, stmt.Get("identify"), vault, path).Scan(&row)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return fmt.Errorf("identify %s: %w", path, err)
		}
		// The full-text index, the index of names and the vector index are all
		// virtual tables, and a cascade reaches none of them.
		if err := chunk.Clear(ctx, tx, row); err != nil {
			return err
		}
		for _, name := range []string{"clear_heading_names", "delete_title_name"} {
			if err := exec(ctx, tx, name, row); err != nil {
				return err
			}
		}
		if err := exec(ctx, tx, "delete_source", row); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// encodeFrontmatter stores what was found as JSON, or nothing when there was no
// frontmatter at all — which is different from an empty one.
//
// YAML can hold things JSON cannot: a NaN, a non-string key, a value that
// refers to itself. The note is still a note, so a failure here comes back as
// a reason, and the caller records it beside the note.
func encodeFrontmatter(n domain.Note) (value any, problem string) {
	if n.Frontmatter == nil {
		return nil, ""
	}
	raw, err := json.Marshal(n.Frontmatter)
	if err != nil {
		return nil, "frontmatter could not be stored: " + err.Error()
	}
	return string(raw), ""
}

// noteType is what the file said it is. A note whose file says nothing is a
// note, and the column holds one of the four words either way.
func noteType(n domain.Note) domain.NoteType {
	if n.Type == "" {
		return domain.TypeNote
	}
	return n.Type
}

// nullable keeps an empty string out of the database, so that "nothing was
// written" and "an empty value was written" stay different questions.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
