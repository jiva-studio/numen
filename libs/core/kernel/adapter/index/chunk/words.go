package chunk

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// FindWords returns the leaf chunks of one vault whose text matches the FTS5 expression, with their negated
// bm25, best first, within q.Paths when set, else within q.PathPrefixes and q.SourceKinds, at most q.Limit.
// An expression FTS5 cannot parse returns source.ErrInvalidWords.
func (q *Search) FindWords(ctx context.Context, vaultID domain.VaultID, query searchquery.WordQuery) ([]searchquery.WordMatch, error) {
	vault, err := getVaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var rows *sql.Rows
	if len(query.Paths) > 0 {
		paths, err := json.Marshal(query.Paths)
		if err != nil {
			return nil, err
		}
		rows, err = q.db.QueryContext(ctx, wordsPathsSQL, query.Expression, vault, string(paths), query.Limit)
		if err != nil {
			return nil, getWordsError(err)
		}
		return scanWordMatches(rows)
	}
	composed, decomposed, err := formatPrefixes(query.PathPrefixes)
	if err != nil {
		return nil, err
	}
	rows, err = q.db.QueryContext(ctx, wordsSQL, query.Expression, vault, composed, decomposed, formatKinds(query.SourceKinds), query.Limit)
	if err != nil {
		return nil, getWordsError(err)
	}
	return scanWordMatches(rows)
}

// GetWordScores returns the negated bm25 of each given chunk whose text matches the expression; a chunk
// that does not match is absent.
func (q *Search) GetWordScores(ctx context.Context, _ domain.VaultID, expression string, chunks []domain.ChunkID) (map[domain.ChunkID]float64, error) {
	if len(chunks) == 0 {
		return map[domain.ChunkID]float64{}, nil
	}
	ids := make([]int64, len(chunks))
	for i, id := range chunks {
		row, err := ParseRow(id)
		if err != nil {
			return nil, err
		}
		ids[i] = row
	}
	rows, err := q.db.QueryContext(ctx, wordScoresSQL, expression, formatInt64s(ids))
	if err != nil {
		return nil, getWordsError(err)
	}
	matches, err := scanWordMatches(rows)
	if err != nil {
		return nil, err
	}
	out := make(map[domain.ChunkID]float64, len(matches))
	for _, m := range matches {
		out[m.Chunk] = m.Score
	}
	return out, nil
}

func scanWordMatches(rows *sql.Rows) (_ []searchquery.WordMatch, err error) {
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	var out []searchquery.WordMatch
	for rows.Next() {
		var row int64
		var score float64
		if err := rows.Scan(&row, &score); err != nil {
			return nil, err
		}
		out = append(out, searchquery.WordMatch{Chunk: FormatID(row), Score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, getWordsError(err)
	}
	return out, nil
}

func getWordsError(err error) error {
	if isQueryError(err) {
		return fmt.Errorf("%w: %w", source.ErrInvalidWords, err)
	}
	return err
}
