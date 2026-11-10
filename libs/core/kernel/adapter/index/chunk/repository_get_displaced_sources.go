// Package chunk stores the chunks a source's text is cut into, and the vectors
// made from them.
//
// The text itself is not stored. A chunk is a place in a file — where it starts
// and how long it is — so showing a passage means reading the source again.
package chunk

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/note/domain"
)
func getDisplacedSources(
	ctx context.Context, tx *sqlwrite.Transaction, vault int64, from, to string,
) (displaced []int64, err error) {
	first, past := getRangeUnder(to)
	rows, err := tx.QueryContext(ctx, getStatement("sources_under"), vault, to, first, past)
	if err != nil {
		return nil, fmt.Errorf("what the vault holds at %s and under it: %w", to, err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	movingFirst, movingPast := getRangeUnder(from)
	for rows.Next() {
		var source int64
		var path string
		if err := rows.Scan(&source, &path); err != nil {
			return nil, fmt.Errorf("what the vault holds at %s and under it: %w", to, err)
		}
		if path == from || (path >= movingFirst && path < movingPast) {
			continue
		}
		displaced = append(displaced, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("what the vault holds at %s and under it: %w", to, err)
	}
	return displaced, nil
}

func updateNoteName(ctx context.Context, tx *sqlwrite.Transaction, vault int64, from, to string) error {
	var title string
	var named bool
	err := tx.QueryRowContext(ctx, getStatement("note_naming"), vault, from).Scan(&title, &named)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("what the note %s is called: %w", from, err)
	}
	displayName, isRenamed := domain.GetNameAfterMove(from, to, title, named)
	if !isRenamed {
		return nil
	}
	if err := exec(ctx, tx, "rename_note", domain.FoldName(domain.Basename(to)), displayName, vault, from); err != nil {
		return err
	}
	return exec(ctx, tx, "rename_title", vault, from, displayName)
}

func getRangeUnder(folder string) (first, past string) {
	return folder + "/", folder + "0"
}

// Clear takes out the chunks of one source, and everything indexed over them.
//
// The rows in the three virtual tables go first, by the chunk's own number.
// Nothing cascades into a virtual table, and a chunk's number is handed to the
func Clear(ctx context.Context, tx *sqlwrite.Transaction, source int64) error {
	for _, name := range []string{"clear_fts", "clear_vec", "clear_sections"} {
		if err := exec(ctx, tx, name, source); err != nil {
			return err
		}
	}
	return exec(ctx, tx, "clear_chunks", source)
}

// Replace makes the chunks of one source the ones given, inside an open transaction.
// A chunk is identified by the hash of its text: a known hash keeps its row and
// vector, a new hash is a new row, a row with no chunk is removed.
// Order matters: new rows first, kept chunks re-pointed at their large chunk,
// removed rows last, because `parent_id ON DELETE CASCADE` removes the chunks inside a large one.
func Replace(ctx context.Context, tx *sqlwrite.Transaction, source, vault int64, chunks []Chunk) error {
	target, err := chunksOf(ctx, tx, source)
	if err != nil {
		return err
	}
	// A chunk with unchanged text keeps its row, so section names are cleared per
	// source and not with the chunks.
	if _, err := tx.ExecContext(ctx, getStatement("clear_sections"), source); err != nil {
		return fmt.Errorf("take the names of this source's sections out of the index: %w", err)
	}

	writer := chunkWriter{tx: tx, target: target, source: source, vault: vault}
	for _, large := range chunks {
		row, err := writer.upsert(ctx, large, nil, len(large.Small) == 0)
		if err != nil {
			return err
		}
		for _, small := range large.Small {
			if _, err := writer.upsert(ctx, small, row, true); err != nil {
				return err
			}
		}
	}
	if err := deleteRows(ctx, tx, target.getUnusedRows()); err != nil {
		return err
	}
	return deleteOrphanVectors(ctx, tx, target.getRemovedHashes())
}

// chunkWriter holds what writing the chunks of one source shares.
type chunkWriter struct {
	tx            *sqlwrite.Transaction
	target        *rows
	source, vault int64
}

func (w chunkWriter) upsert(ctx context.Context, c Chunk, parent any, isLeaf bool) (int64, error) {
	tx, target, source, vault := w.tx, w.target, w.source, w.vault
	key := textID{hash: computeHash(c.Text), isSmall: parent != nil}
	if row, hasRow := target.takeUnusedRow(key); hasRow {
		if _, err := tx.ExecContext(ctx, getStatement("move_chunk"),
			c.Start, c.Length, parent, newNullable(c.Location), row); err != nil {
			return 0, fmt.Errorf("move a chunk to where its text now is: %w", err)
		}
		if err := writeSectionNames(ctx, tx, row, c); err != nil {
			return 0, err
		}
		return row, nil
	}

	var row int64
	err := tx.QueryRowContext(ctx, getStatement("insert_chunk"),
		source, vault, c.Start, c.Length, parent, newNullable(c.Location), key.hash).Scan(&row)
	if err != nil {
		return 0, fmt.Errorf("store a chunk of this source: %w", err)
	}
	if isLeaf {
		if _, err := tx.ExecContext(ctx, getStatement("insert_fts"), row, c.Text); err != nil {
			return 0, fmt.Errorf("index a chunk for the words in it: %w", err)
		}
	}
	if err := writeSectionNames(ctx, tx, row, c); err != nil {
		return 0, err
	}
	return row, nil
}

func writeSectionNames(ctx context.Context, tx *sqlwrite.Transaction, row int64, c Chunk) error {
	if len(c.Opens) == 0 {
		return nil
	}
	if _, err := tx.ExecContext(ctx, getStatement("insert_section"), row, strings.Join(c.Opens, "\n")); err != nil {
		return fmt.Errorf("index the names of the sections a chunk opens: %w", err)
	}
	return nil
}

// textID is what a chunk has to hold to be held on a row: the same text, cut at
// two sizes are separate populations.
type textID struct {
	hash    string
	isSmall bool
}

// rows holds the existing rows of a source, indexed for matching against new chunks.
type rows struct {
	candidates    map[textID][]int64
	remainingErrs map[int64]bool
	hash          map[int64]string
}

func chunksOf(ctx context.Context, tx *sqlwrite.Transaction, source int64) (_ *rows, err error) {
	cursor, err := tx.QueryContext(ctx, getStatement("chunks_of"), source)
	if err != nil {
		return nil, fmt.Errorf("the chunks this source is already cut into: %w", err)
	}
	defer func() { err = sqlwrite.JoinClose(err, cursor.Close()) }()

	h := &rows{candidates: map[textID][]int64{}, remainingErrs: map[int64]bool{}, hash: map[int64]string{}}
	for cursor.Next() {
		var row int64
		var key textID
		if err := cursor.Scan(&row, &key.hash, &key.isSmall); err != nil {
			return nil, fmt.Errorf("the chunks this source is already cut into: %w", err)
		}
		h.candidates[key] = append(h.candidates[key], row)
		h.remainingErrs[row] = true
		h.hash[row] = key.hash
	}
	return h, cursor.Err()
}

func (h *rows) takeUnusedRow(key textID) (int64, bool) {
	rows := h.candidates[key]
	if len(rows) == 0 {
		return 0, false
	}
	row := rows[0]
	h.candidates[key] = rows[1:]
	delete(h.remainingErrs, row)
	return row, true
}

func (h *rows) getUnusedRows() []int64 {
	out := make([]int64, 0, len(h.remainingErrs))
	for row := range h.remainingErrs {
		out = append(out, row)
	}
	slices.Sort(out)
	return out
}

func deleteRows(ctx context.Context, tx *sqlwrite.Transaction, rows []int64) error {
	for _, row := range rows {
		for _, name := range []string{"delete_fts", "delete_vec", "delete_chunk"} {
			if err := exec(ctx, tx, name, row); err != nil {
				return err
			}
		}
	}
	return nil
}

func computeHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func exec(ctx context.Context, tx *sqlwrite.Transaction, name string, args ...any) error {
	if _, err := tx.ExecContext(ctx, getStatement(name), args...); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("%s: %w", name, ctx.Err())
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// querier runs a single-row query, on a transaction or on a plain connection.
type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func vaultRow(ctx context.Context, db querier, identifier domain.VaultID) (int64, error) {
	var row int64
	err := db.QueryRowContext(ctx, vaultRowSQL, string(identifier)).Scan(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errNoVault
	}
	return row, err
}

func getSourceRow(ctx context.Context, db querier, vault int64, kind, path string) (int64, error) {
	var row int64
	err := db.QueryRowContext(ctx, identifySQL, vault, kind, path).Scan(&row)
	return row, err
}

func newNullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (h *rows) getRemovedHashes() []string {
	out := make([]string, 0, len(h.remainingErrs))
	for row := range h.remainingErrs {
		if hash := h.hash[row]; hash != "" {
			out = append(out, hash)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
