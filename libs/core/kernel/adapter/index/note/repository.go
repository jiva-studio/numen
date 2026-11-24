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

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/changefeed"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

//go:embed sql/*.sql
var files embed.FS

func getStatement(name string) string { return sqlfile.Read(files, "sql", name) }

// kind is what a note is among the sources the index holds. Everything this
// package writes and reads is one, and a path naming a file of another kind
// answers nothing here.
const kind = "note"

// Repository is the collection of notes. It puts one in and takes one out, and
// answers no questions about them.
type Repository struct {
	db      *sql.DB
	changes *FrontmatterChanges
	feed    *changefeed.Changes
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

// SetFrontmatterChanges returns the repository that hands the paths of the notes whose frontmatter a write
// changed to changes.
func (r *Repository) SetFrontmatterChanges(changes *FrontmatterChanges) *Repository {
	return &Repository{db: r.db, changes: changes, feed: r.feed}
}

// SetChangeFeed returns the repository that tells feed of each committed write that can change the file
// change feed.
func (r *Repository) SetChangeFeed(feed *changefeed.Changes) *Repository {
	return &Repository{db: r.db, changes: r.changes, feed: feed}
}

func exec(ctx context.Context, tx *sqlwrite.Transaction, name string, args ...any) error {
	if _, err := tx.ExecContext(ctx, getStatement(name), args...); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// Save writes a group of notes in one transaction.
//
// A note and the size and date that call it up to date are stored together or
// not at all, so an interrupted scan leaves files to be read again.
func (r *Repository) Save(
	ctx context.Context, vaultID domain.VaultID, notes []domain.NoteChunks,
) error {
	if len(notes) == 0 {
		return nil
	}
	var changed []string
	err := sqlwrite.RunInTransaction(ctx, r.db, func(tx *sqlwrite.Transaction) error {
		changed = nil
		vault, err := getVaultRow(ctx, tx, vaultID)
		if err != nil {
			return err
		}
		for _, one := range notes {
			isChanged, err := saveNote(ctx, tx, vault, one)
			if err != nil {
				return fmt.Errorf("%s: %w", one.Note.FileStat.Path, err)
			}
			if isChanged {
				changed = append(changed, one.Note.FileStat.Path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	r.feed.Publish(vaultID)
	r.changes.publish(vaultID, changed)
	return nil
}

func saveNote(
	ctx context.Context, tx *sqlwrite.Transaction, vault int64, indexed domain.NoteChunks,
) (isFrontmatterChanged bool, err error) {
	n := indexed.Note
	previous, err := readStoredFrontmatter(ctx, tx, vault, n.FileStat.Path)
	if err != nil {
		return false, err
	}
	before, err := changefeed.ReadState(ctx, tx, vault, n.FileStat.Path)
	if err != nil {
		return false, err
	}
	row, stored, err := saveSourceAndNote(ctx, tx, vault, indexed)
	if err != nil {
		return false, err
	}
	if err := changefeed.RecordSaved(ctx, tx, vault, n.FileStat.Path, before); err != nil {
		return false, err
	}
	isFrontmatterChanged = previous != stored
	if isFrontmatterChanged {
		if err := changefeed.RecordNoteChange(ctx, tx, vault, row, manifest.EventNoteFrontmatterChanged, before.NoteType); err != nil {
			return false, err
		}
	}
	return isFrontmatterChanged, saveNoteParts(ctx, tx, vault, row, indexed)
}

func saveNoteParts(ctx context.Context, tx *sqlwrite.Transaction, vault, row int64, indexed domain.NoteChunks) error {
	n := indexed.Note
	// A chunk whose text is what it was keeps its row, and the vector made from it.
	if err := chunk.Replace(ctx, tx, row, vault, chunk.ConvertChunks(indexed.Chunks)); err != nil {
		return err
	}
	for _, h := range indexed.Headings {
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
	if err := insertLinks(ctx, tx, row, n.Links); err != nil {
		return err
	}
	for _, detail := range n.Problems {
		if err := exec(ctx, tx, "insert_problem", row, detail); err != nil {
			return err
		}
	}
	return nil
}

func saveSourceAndNote(
	ctx context.Context, tx *sqlwrite.Transaction, vault int64, indexed domain.NoteChunks,
) (row int64, stored sql.NullString, err error) {
	n := indexed.Note
	frontmatter, storeErr := encodeFrontmatter(n)
	problem := n.FrontmatterErr
	if storeErr != "" {
		// Frontmatter that cannot be stored leaves the note indexed without it,
		// and the reason is recorded as a problem of the note.
		problem = strings.TrimSpace(problem + "\n" + storeErr)
	}

	if text, ok := frontmatter.(string); ok {
		stored = sql.NullString{String: text, Valid: true}
	}
	var updated bool
	if err := tx.QueryRowContext(ctx, getStatement("save_source"),
		vault, n.FileStat.Path, kind, n.FileStat.Size, chunk.ConvertTimeToNanos(n.FileStat.ModTime),
		newNullableValue(n.FileStat.Hash), newNullableValue(indexed.Recipe), nil,
	).Scan(&row, &updated); err != nil {
		return 0, stored, fmt.Errorf("record the source this note is: %w", err)
	}
	if err := exec(ctx, tx, "save_note", row, vault, domain.NormalizeName(domain.GetBasename(n.FileStat.Path)),
		n.Title, string(getNoteType(n)), newNullableValue(n.ID), frontmatter, newNullableValue(problem)); err != nil {
		return 0, stored, err
	}
	if !updated {
		return row, stored, nil
	}
	for _, name := range []string{
		"clear_heading_names", "clear_headings", "clear_links", "clear_problems",
	} {
		if err := exec(ctx, tx, name, row); err != nil {
			return 0, stored, err
		}
	}
	return row, stored, nil
}

func readStoredFrontmatter(ctx context.Context, tx *sqlwrite.Transaction, vault int64, path string) (sql.NullString, error) {
	var stored sql.NullString
	err := tx.QueryRowContext(ctx, getStatement("frontmatter_at"), vault, path).Scan(new(int64), &path, &stored)
	if errors.Is(err, sql.ErrNoRows) {
		return sql.NullString{}, nil
	}
	if err != nil {
		return sql.NullString{}, fmt.Errorf("read the stored frontmatter of %s: %w", path, err)
	}
	return stored, nil
}

func insertLinks(ctx context.Context, tx *sqlwrite.Transaction, row int64, links []domain.Link) error {
	for i, l := range links {
		if _, err := tx.ExecContext(ctx, getStatement("insert_link"), row, i,
			l.Target.Scheme, l.Target.Value, domain.NormalizeName(domain.LinkName(l.Target.Value)),
			string(l.Role), newNullableValue(l.Type), newNullableValue(l.Why), newNullableValue(l.Label),
			newNullableValue(string(l.Marker)),
		); err != nil {
			return fmt.Errorf("store what this note points at: %w", err)
		}
	}
	return nil
}

func (r *Repository) Remove(ctx context.Context, vaultID domain.VaultID, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	var changed []string
	err := sqlwrite.RunInTransaction(ctx, r.db, func(tx *sqlwrite.Transaction) error {
		changed = nil
		vault, err := getVaultRow(ctx, tx, vaultID)
		if err != nil {
			return err
		}
		for _, path := range paths {
			previous, err := readStoredFrontmatter(ctx, tx, vault, path)
			if err != nil {
				return err
			}
			if err := removeNote(ctx, tx, vault, path); err != nil {
				return err
			}
			if previous.Valid {
				changed = append(changed, path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	r.feed.Publish(vaultID)
	r.changes.publish(vaultID, changed)
	return nil
}

func removeNote(ctx context.Context, tx *sqlwrite.Transaction, vault int64, path string) error {
	var row int64
	err := tx.QueryRowContext(ctx, getStatement("identify"), vault, path).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("identify %s: %w", path, err)
	}
	if err := chunk.Clear(ctx, tx, row); err != nil {
		return err
	}
	for _, name := range []string{"clear_heading_names", "delete_title_name"} {
		if err := exec(ctx, tx, name, row); err != nil {
			return err
		}
	}
	if err := changefeed.RecordRemoved(ctx, tx, vault, row); err != nil {
		return err
	}
	return exec(ctx, tx, "delete_source", row)
}

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

func getNoteType(n domain.Note) domain.NoteType {
	if n.Type == "" {
		return domain.TypeNote
	}
	return n.Type
}

func newNullableValue(s string) any {
	if s == "" {
		return nil
	}
	return s
}
