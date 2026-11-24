// Package changefeed keeps the file change feed of each vault: one row for each event of each source,
// numbered by the write that stored it last. A write of the index records its changes in its own
// transaction, so the feed holds what the index holds.
package changefeed

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/filechange"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

//go:embed sql/*.sql
var files embed.FS

func getStatement(name string) string { return sqlfile.Read(files, "sql", name) }

// RemovalLifetime is how long a removal stays in the feed.
const RemovalLifetime = 183 * 24 * time.Hour

// State is the source a vault holds at a path before a write, with the size, modification time and
// hash a later write is compared with.
type State struct {
	ID         int64
	Size       int64
	ModifiedAt int64
	Hash       string
	NoteType   domain.NoteType
	IsFound    bool
}

// ReadState returns the source the vault holds at the path.
func ReadState(ctx context.Context, tx *sqlwrite.Transaction, vault int64, path string) (State, error) {
	var s State
	err := tx.QueryRowContext(ctx, getStatement("source_state"), vault, path).Scan(&s.ID, &s.Size, &s.ModifiedAt, &s.Hash, &s.NoteType)
	if errors.Is(err, sql.ErrNoRows) {
		return State{}, nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read the source at %s: %w", path, err)
	}
	s.IsFound = true
	return s, nil
}

// RecordSaved records file.added for a source the write created at the path and file.changed for one
// whose size, modification time or hash it changed. A hash that was not known before, or that the write
// leaves unknown, is not a change.
func RecordSaved(ctx context.Context, tx *sqlwrite.Transaction, vault int64, path string, before State) error {
	after, err := ReadState(ctx, tx, vault, path)
	if err != nil || !after.IsFound {
		return err
	}
	switch {
	case !before.IsFound || before.ID != after.ID:
		return Record(ctx, tx, vault, after.ID, manifest.EventFileAdded)
	case before.Size != after.Size || before.ModifiedAt != after.ModifiedAt || isHashChanged(before.Hash, after.Hash):
		return RecordNoteChange(ctx, tx, vault, after.ID, manifest.EventFileChanged, before.NoteType)
	}
	return nil
}

func isHashChanged(before, after string) bool {
	return before != "" && after != "" && before != after
}

// Record stores the event of the source as its latest, under the next number of the vault's feed.
func Record(ctx context.Context, tx *sqlwrite.Transaction, vault, source int64, event string) error {
	return RecordNoteChange(ctx, tx, vault, source, event, "")
}

// RecordNoteChange includes the previous note type among the types affected by the event.
func RecordNoteChange(ctx context.Context, tx *sqlwrite.Transaction, vault, source int64, event string, previous domain.NoteType) error {
	return record(ctx, tx, vault, source, event, noteTypeHistory{previous: previous, sequences: "{}"})
}

// noteTypeHistory holds the former type of a retyped note and the type sequences retained on removal.
type noteTypeHistory struct {
	previous  domain.NoteType
	sequences string
}

func record(ctx context.Context, tx *sqlwrite.Transaction, vault, source int64, event string, history noteTypeHistory) error {
	seq, err := getNextSeq(ctx, tx, vault)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, getStatement("save_change"), vault, source, event, seq, string(history.previous), history.sequences); err != nil {
		return fmt.Errorf("record %s of source %d: %w", event, source, err)
	}
	return nil
}

// RecordRemoved replaces the events of the source with its removal.
func RecordRemoved(ctx context.Context, tx *sqlwrite.Transaction, vault, source int64) error {
	var history string
	if err := tx.QueryRowContext(ctx, getStatement("note_type_history"), vault, source, manifest.EventFileRemoved).Scan(&history); err != nil {
		return fmt.Errorf("read note type history of source %d: %w", source, err)
	}
	if _, err := tx.ExecContext(ctx, getStatement("clear_changes"), vault, source); err != nil {
		return fmt.Errorf("clear the changes of source %d: %w", source, err)
	}
	return record(ctx, tx, vault, source, manifest.EventFileRemoved, noteTypeHistory{sequences: history})
}

func getNextSeq(ctx context.Context, tx *sqlwrite.Transaction, vault int64) (int64, error) {
	var seq int64
	err := tx.QueryRowContext(ctx, getStatement("next_seq"), vault).Scan(&seq)
	if errors.Is(err, sql.ErrNoRows) {
		if err := OpenFeed(ctx, tx, vault); err != nil {
			return 0, err
		}
		err = tx.QueryRowContext(ctx, getStatement("next_seq"), vault).Scan(&seq)
	}
	if err != nil {
		return 0, fmt.Errorf("number the next change: %w", err)
	}
	return seq, nil
}

// OpenFeed gives the vault a feed of an identity no other feed has, unless it has one.
func OpenFeed(ctx context.Context, tx *sqlwrite.Transaction, vault int64) error {
	if _, err := tx.ExecContext(ctx, getStatement("open_feed"), vault); err != nil {
		return fmt.Errorf("open the change feed: %w", err)
	}
	return nil
}

// Trim drops the removals stored longer ago than the age and raises the number each feed dropped removals up to.
func Trim(ctx context.Context, db *sql.DB, age time.Duration) error {
	return sqlwrite.RunInTransaction(ctx, db, func(tx *sqlwrite.Transaction) error {
		for _, name := range []string{"trim_seq", "trim"} {
			if _, err := tx.ExecContext(ctx, getStatement(name), manifest.EventFileRemoved, age.Nanoseconds()); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
		return nil
	})
}

// Feed reads the change feeds.
type Feed struct{ db *sql.DB }

func NewFeed(db *sql.DB) *Feed { return &Feed{db: db} }

// GetHead returns the state of the feed of the vault: the zero head for a vault the index does not hold.
func (f *Feed) GetHead(ctx context.Context, vaultID domain.VaultID) (filechange.Head, error) {
	var head filechange.Head
	err := f.db.QueryRowContext(ctx, getStatement("head"), string(vaultID)).
		Scan(&head.Cursor.FeedID, &head.Cursor.Seq, &head.TrimmedSeq)
	if errors.Is(err, sql.ErrNoRows) {
		return filechange.Head{}, nil
	}
	if err != nil {
		return filechange.Head{}, fmt.Errorf("read the change feed: %w", err)
	}
	return head, nil
}

// ListFileChanges returns the changes of the events after the number after and up to the number upTo,
// at most limit, in the order they were stored.
func (f *Feed) ListFileChanges(
	ctx context.Context, vaultID domain.VaultID, events []string, after, upTo int64, limit int,
) (_ []filechange.Change, err error) {
	names, err := json.Marshal(events)
	if err != nil {
		return nil, err
	}
	rows, err := f.db.QueryContext(ctx, getStatement("list"), manifest.EventFileRemoved, string(vaultID), after, upTo, string(names), limit)
	if err != nil {
		return nil, fmt.Errorf("list the file changes: %w", err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	var out []filechange.Change
	for rows.Next() {
		var c filechange.Change
		var noteTypes []byte
		if err := rows.Scan(&c.SourceID, &c.Path, &c.Event, &c.Seq, &c.Hash, &c.Size, &c.ModifiedAt, &noteTypes); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(noteTypes, &c.NoteTypes); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListChangedFiles returns each affected path and note type once in the requested sequence range.
// A removal retains the path of the removed file.
func (f *Feed) ListChangedFiles(ctx context.Context, vaultID domain.VaultID, after, upTo int64) (_ []filechange.File, err error) {
	rows, err := f.db.QueryContext(ctx, getStatement("changed_paths"), manifest.EventFileRemoved, string(vaultID), after, upTo)
	if err != nil {
		return nil, fmt.Errorf("list the changed files: %w", err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	var out []filechange.File
	for rows.Next() {
		var file filechange.File
		if err := rows.Scan(&file.Path, &file.NoteType); err != nil {
			return nil, err
		}
		out = append(out, file)
	}
	return out, rows.Err()
}
