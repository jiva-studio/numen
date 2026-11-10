package chunk

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/index"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// GetIndexCounts returns the chunks, vectors and sources of a vault, and the sources by extension.
// A vault the index does not know has none.
func (q *Vectors) GetIndexCounts(ctx context.Context, vaultID domain.VaultID, recipe string) (_ index.Counts, err error) {
	vault, err := vaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return index.Counts{}, nil
	}
	if err != nil {
		return index.Counts{}, err
	}
	chunks, vectors, err := q.GetProgress(ctx, vaultID, recipe)
	if err != nil {
		return index.Counts{}, err
	}
	extensions, err := q.getExtensions(ctx, vault)
	if err != nil {
		return index.Counts{}, err
	}
	counts := index.Counts{Chunks: chunks, Vectors: vectors, Sources: 0, Extensions: extensions}
	for _, one := range extensions {
		counts.Sources += one.Sources
	}
	return counts, nil
}

func (q *Vectors) getExtensions(ctx context.Context, vault int64) (_ []index.ExtensionCount, err error) {
	rows, err := q.db.QueryContext(ctx, extensionsSQL, vault, vault, q.minimum.Get())
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, rows.Close()) }()

	var out []index.ExtensionCount
	for rows.Next() {
		var one index.ExtensionCount
		if err := rows.Scan(&one.Extension, &one.Sources, &one.Chunks); err != nil {
			return nil, err
		}
		out = append(out, one)
	}
	return out, rows.Err()
}

// GetIndexSizes returns the size of the database and of the tables that hold vectors.
func (q *Vectors) GetIndexSizes(ctx context.Context) (index.Sizes, error) {
	ctx, err := sqlwrite.Uninterrupted(ctx)
	if err != nil {
		return index.Sizes{}, err
	}
	var sizes index.Sizes
	err = q.db.QueryRowContext(ctx, sizesSQL).Scan(&sizes.Database, &sizes.Vectors)
	return sizes, err
}
