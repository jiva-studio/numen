package chunk

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/port/indexport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// GetIndexCounts returns the chunks, vectors and sources of a vault, and the sources by extension.
// A vault the index does not know has none.
func (q *Queries) GetIndexCounts(ctx context.Context, vaultID domain.VaultID, recipe string) (_ indexport.IndexCounts, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return indexport.IndexCounts{}, nil
	}
	if err != nil {
		return indexport.IndexCounts{}, err
	}
	chunks, vectors, err := q.GetProgress(ctx, vaultID, recipe)
	if err != nil {
		return indexport.IndexCounts{}, err
	}
	extensions, err := q.getExtensions(ctx, vault)
	if err != nil {
		return indexport.IndexCounts{}, err
	}
	counts := indexport.IndexCounts{Chunks: chunks, Vectors: vectors, Sources: 0, Extensions: extensions}
	for _, one := range extensions {
		counts.Sources += one.Sources
	}
	return counts, nil
}

func (q *Queries) getExtensions(ctx context.Context, vault int64) (_ []indexport.ExtensionCount, err error) {
	rows, err := q.db.QueryContext(ctx, extensionsSQL, vault, vault, q.minimum.Get())
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []indexport.ExtensionCount
	for rows.Next() {
		var one indexport.ExtensionCount
		if err := rows.Scan(&one.Extension, &one.Sources, &one.Chunks); err != nil {
			return nil, err
		}
		out = append(out, one)
	}
	return out, rows.Err()
}

// GetIndexSizes returns the size of the database and of the tables that hold vectors.
func (q *Queries) GetIndexSizes(ctx context.Context) (indexport.IndexSizes, error) {
	ctx, err := sqlwrite.Uninterrupted(ctx)
	if err != nil {
		return indexport.IndexSizes{}, err
	}
	var sizes indexport.IndexSizes
	err = q.db.QueryRowContext(ctx, sizesSQL).Scan(&sizes.Database, &sizes.Vectors)
	return sizes, err
}
