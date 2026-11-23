package chunk

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ListDocuments returns the sources of one vault under any of the prefixes, of the source kinds and at the paths
// of the query, with their note title and type, the time of their last change, their size and the count of their
// small chunks with a vector under the recipe, in order of path, at most q.Limit+1 of them; all of them when
// q.Limit is zero or below.
func (q *Search) ListDocuments(ctx context.Context, vaultID domain.VaultID, query searchquery.DocumentQuery) (_ []searchquery.Document, err error) {
	vault, err := getVaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	composed, decomposed, err := formatPrefixes(query.PathPrefixes)
	if err != nil {
		return nil, err
	}
	paths, err := json.Marshal(append([]string{}, query.Paths...))
	if err != nil {
		return nil, fmt.Errorf("encode the paths: %w", err)
	}
	limit := -1
	if query.Limit > 0 {
		limit = query.Limit + 1
	}
	isWithinPaths := 0
	if query.IsWithinPaths {
		isWithinPaths = 1
	}
	rows, err := q.db.QueryContext(ctx, documentsSQL, vault, composed, decomposed, formatKinds(query.SourceKinds), query.Recipe, limit,
		isWithinPaths, string(paths))
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	var out []searchquery.Document
	for rows.Next() {
		var d searchquery.Document
		var modified int64
		if err := rows.Scan(&d.Path, &d.SourceKind, &d.Title, &d.NoteType, &modified, &d.Size, &d.ChunkCount); err != nil {
			return nil, err
		}
		d.ModifiedAt = ConvertNanosToTime(modified)
		out = append(out, d)
	}
	return out, rows.Err()
}

// ListDocumentVectors returns the vectors under the recipe of the small chunks of the sources of one vault
// at the paths, at most q.Limit after the chunk q.After, in order of chunk id.
func (q *Search) ListDocumentVectors(ctx context.Context, vaultID domain.VaultID, query searchquery.DocumentVectorQuery) (_ []source.StoredVector, err error) {
	if len(query.Paths) == 0 {
		return nil, nil
	}
	vault, err := getVaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	after := int64(0)
	if query.After != "" {
		if after, err = ParseRow(query.After); err != nil {
			return nil, err
		}
	}
	paths, err := json.Marshal(query.Paths)
	if err != nil {
		return nil, err
	}
	rows, err := q.db.QueryContext(ctx, documentVectorsSQL, vault, string(paths), query.Recipe, after, query.Limit)
	if err != nil {
		return nil, err
	}
	return scanStoredVectors(rows)
}

func scanStoredVectors(rows *sql.Rows) (_ []source.StoredVector, err error) {
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()
	var out []source.StoredVector
	for rows.Next() {
		var row int64
		var stored []byte
		if err := rows.Scan(&row, &stored); err != nil {
			return nil, err
		}
		out = append(out, source.StoredVector{Chunk: FormatID(row), Vector: stored})
	}
	return out, rows.Err()
}

func formatPrefixes(prefixes []string) (composed, decomposed string, err error) {
	nfc, nfd := make([]string, len(prefixes)), make([]string, len(prefixes))
	for i, prefix := range prefixes {
		nfc[i], nfd[i] = norm.NFC.String(prefix), norm.NFD.String(prefix)
	}
	first, err := json.Marshal(nfc)
	if err != nil {
		return "", "", fmt.Errorf("encode the path prefixes: %w", err)
	}
	second, err := json.Marshal(nfd)
	if err != nil {
		return "", "", fmt.Errorf("encode the path prefixes: %w", err)
	}
	return string(first), string(second), nil
}
