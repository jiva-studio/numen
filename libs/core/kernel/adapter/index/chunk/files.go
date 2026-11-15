package chunk

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Files answers what the index holds about the sources of a vault: their stats, their text and where a
// chunk is read from.
type Files struct{ db *sql.DB }

// NewFiles creates the queries about the sources over the read pool.
func NewFiles(db *sql.DB) *Files { return &Files{db: db} }

// GetFileStats is what the index believes about each file of one kind, keyed by
// path, so a scan can decide what to read again without opening anything.
func (q *Files) GetFileStats(ctx context.Context, vaultID domain.VaultID, kind string) (_ map[string]domain.FileStat, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return map[string]domain.FileStat{}, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, fileStatsSQL, vault, kind)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	out := map[string]domain.FileStat{}
	for rows.Next() {
		var ref domain.FileStat
		var mtime int64
		if err := rows.Scan(&ref.Path, &ref.Size, &mtime); err != nil {
			return nil, err
		}
		ref.ModTime = ConvertNanosToTime(mtime)
		out[ref.Path] = ref
	}
	return out, rows.Err()
}

// GetSourcesUnder is every source the vault holds at a path and beneath it, by
// path: the one file, or everything a folder holds.
func (q *Files) GetSourcesUnder(ctx context.Context, vaultID domain.VaultID, path string) (_ []domain.FileStat, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	first, past := getRangeUnder(path)
	rows, err := q.db.QueryContext(ctx, fileStatsUnderSQL, vault, path, vault, first, past)
	if err != nil {
		return nil, fmt.Errorf("what the vault holds at %s and under it: %w", path, err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []domain.FileStat
	for rows.Next() {
		var ref domain.FileStat
		var mtime int64
		if err := rows.Scan(&ref.Path, &ref.Kind, &ref.Size, &mtime); err != nil {
			return nil, err
		}
		ref.ModTime = ConvertNanosToTime(mtime)
		out = append(out, ref)
	}
	return out, rows.Err()
}

// GetRecord is where one chunk's text is read from. False when the vault holds no
// such chunk.
func (q *Files) GetRecord(ctx context.Context, vaultID domain.VaultID, chunk int64) (Record, bool, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	p, found, err := scanRecord(q.db.QueryRowContext(ctx, recordSQL, chunk, vault), chunk)
	return p, found, err
}

func scanRecord(row *sql.Row, chunk int64) (Record, bool, error) {
	p := Record{Chunk: chunk}
	err := row.Scan(&p.Path, &p.Producer, &p.ContentHash, &p.Start, &p.Length, &p.Location, &p.Parent)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, false, nil
	}
	if err != nil {
		return Record{}, false, err
	}
	return p, true, nil
}

// GetUnchunkedSources is the sources of one kind with no small chunk: the file
// changed, or it has never been cut.
func (q *Files) GetUnchunkedSources(ctx context.Context, vaultID domain.VaultID, kind string, limit int) ([]string, error) {
	return q.paths(ctx, vaultID, "unchunked", unchunkedSQL, limit, func(vault int64) []any {
		return []any{vault, kind, limit}
	})
}

// GetSourcesWithOtherRecipe is the sources of one kind whose text was not extracted by any
// of the recipes now in use.
func (q *Files) GetSourcesWithOtherRecipe(ctx context.Context, vaultID domain.VaultID, kind string, recipes []string, limit int) ([]string, error) {
	named, err := json.Marshal(recipes)
	if err != nil {
		return nil, err
	}
	return q.paths(ctx, vaultID, "stale_recipe", staleRecipeSQL, limit, func(vault int64) []any {
		return []any{vault, kind, string(named), limit}
	})
}

func (q *Files) paths(ctx context.Context, vaultID domain.VaultID, name, query string, limit int, args func(vault int64) []any) (_ []string, err error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%s needs a positive limit, got %d", name, limit)
	}
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := q.db.QueryContext(ctx, query, args(vault)...)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		out = append(out, path)
	}
	return out, rows.Err()
}

// SourceText is one source whose text a producer made: where the file is, what
// made the text, and the hash the files of that recognition are kept under.
type SourceText struct {
	Path        string
	Producer    string
	ContentHash string
	Size        int64
	MTime       int64
}

// GetSourceText is what one source's text came from, and false where the index holds
// no source at that path.
func (q *Files) GetSourceText(ctx context.Context, vaultID domain.VaultID, path string) (SourceText, bool, error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return SourceText{}, false, nil
	}
	if err != nil {
		return SourceText{}, false, err
	}
	found := SourceText{Path: path}
	err = q.db.QueryRowContext(ctx, sourceTextSQL, vault, path).Scan(&found.Producer, &found.ContentHash, &found.Size, &found.MTime)
	if errors.Is(err, sql.ErrNoRows) {
		return SourceText{}, false, nil
	}
	if err != nil {
		return SourceText{}, false, fmt.Errorf("reading %s: %w", path, err)
	}
	return found, true, nil
}

// GetRecognizedSources returns, by path, the sources of one kind whose text a
// producer created and not read from the file's own bytes.
// A scan uses it to find sources whose files are gone: the store is a folder
// the user may empty.
// search results.
func (q *Files) GetRecognizedSources(ctx context.Context, vaultID domain.VaultID, kind string) (_ []SourceText, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, recognizedSQL, vault, kind)
	if err != nil {
		return nil, fmt.Errorf("what a model made the text of: %w", err)
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []SourceText
	for rows.Next() {
		var r SourceText
		if err := rows.Scan(&r.Path, &r.Producer, &r.ContentHash); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
