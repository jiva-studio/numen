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

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index/chunk"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/window"
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

	// sizes are what a note is cut at. A repository told none cuts at the sizes
	// the window package names.
	sizes window.Sizes
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// Cut is the repository, cutting a note at the sizes given. The settings decide
// them, and what has read the settings passes them in here.
func (r *Repository) Cut(sizes window.Sizes) *Repository {
	return &Repository{db: r.db, sizes: sizes}
}

// exec runs a named statement and says which one failed. A bare driver error
// from one of the many statements in a transaction is a schema mistake nobody
// can locate.
func exec(ctx context.Context, tx *sql.Tx, name string, args ...any) error {
	if _, err := tx.ExecContext(ctx, stmt.Get(name), args...); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// Save writes a group of notes in one transaction.
//
// A note and the size and date that call it up to date are stored together or
// not at all, so an interrupted scan leaves files to be read again.
func (r *Repository) Save(ctx context.Context, vaultID string, notes []domain.Note) error {
	if len(notes) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	vault, err := vaultRow(ctx, tx, vaultID)
	if err != nil {
		return err
	}
	for _, n := range notes {
		if err := saveNote(ctx, tx, vault, n, r.sizes); err != nil {
			return fmt.Errorf("%s: %w", n.Ref.Path, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func saveNote(ctx context.Context, tx *sql.Tx, vault int64, n domain.Note, sizes window.Sizes) error {
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
	if err := tx.QueryRowContext(ctx, stmt.Get("save"),
		vault, n.Ref.Path, kind, n.Ref.Size, n.Ref.MTime).Scan(&row); err != nil {
		return fmt.Errorf("save: %w", err)
	}
	if err := exec(ctx, tx, "save_note", row, vault, domain.Basename(n.Ref.Path),
		n.Title, nullable(n.ID), frontmatter, nullable(problem)); err != nil {
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
	// The note goes in as its own large window, so the words in it are findable
	// as soon as it is indexed. A window whose text is what it was keeps its
	// row, and the vector made from it.
	if err := chunk.Replace(ctx, tx, row, vault, cut(n, sizes)); err != nil {
		return err
	}
	for _, h := range n.Headings {
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
	if len(n.Links) > 0 {
		insert, err := tx.PrepareContext(ctx, stmt.Get("insert_link"))
		if err != nil {
			return fmt.Errorf("insert_link: %w", err)
		}
		defer insert.Close()
		for i, l := range n.Links {
			if _, err := insert.ExecContext(ctx, row, i,
				l.Target.Scheme, l.Target.Value, domain.Basename(l.Target.Value),
				string(l.Role), nullable(l.Type), nullable(l.Note), nullable(l.Label),
			); err != nil {
				return fmt.Errorf("insert_link: %w", err)
			}
		}
	}
	for _, detail := range n.Problems {
		if err := exec(ctx, tx, "insert_problem", row, detail); err != nil {
			return err
		}
	}
	return nil
}

// cut is how a note is cut: one large window over the whole of it, and the small
// windows inside that carry the vectors. The small windows are cut at the sizes
// a book's are, and a mixed vault ranks by what a passage says.
//
// The large window is the note itself. The sizes decide the small windows inside
// it.
//
// The note's headings are its places, so the small windows of one section are
// tiled inside that section and no window runs across a heading.
//
// Offsets are into the file. The body begins after the frontmatter, and every
// window is moved out by as much.
func cut(n domain.Note, sizes window.Sizes) []chunk.Window {
	at := int(n.Ref.Size) - len(n.Body)
	if at < 0 {
		at = 0
	}
	sizes.Large = window.Whole

	out := make([]chunk.Window, 0, 1)
	for _, large := range window.Cut(n.Body, places(n), sizes) {
		// The title is searched together with the body: a note is looked for by
		// the name it was given.
		w := chunk.Window{
			Start:    at + large.Start,
			Length:   large.Length,
			Location: large.Location,
			Text:     n.Title + "\n" + large.Slice(n.Body),
		}
		for _, small := range large.Small {
			w.Small = append(w.Small, chunk.Window{
				Start:    at + small.Start,
				Length:   small.Length,
				Location: small.Location,
				Text:     small.Slice(n.Body),
			})
		}
		out = append(out, w)
	}
	if len(out) == 0 {
		// A note of a title and no words is answered by its title.
		out = append(out, chunk.Window{Start: at, Length: len(n.Body), Text: n.Title})
	}
	return out
}

// places is where a note names the section that follows. A heading carries the
// byte its line begins at in the body, which is the offset a window is cut
// against, and the heading's own text is what the section is called.
func places(n domain.Note) []window.Place {
	if len(n.Headings) == 0 {
		return nil
	}
	out := make([]window.Place, 0, len(n.Headings))
	for _, h := range n.Headings {
		out = append(out, window.Place{Title: h.Text, Offset: h.Offset})
	}
	return out
}

func (r *Repository) Remove(ctx context.Context, vaultID string, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
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
		if err := exec(ctx, tx, "delete", row); err != nil {
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

// nullable keeps an empty string out of the database, so that "nothing was
// written" and "an empty value was written" stay different questions.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
