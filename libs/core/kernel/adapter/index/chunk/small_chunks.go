package chunk

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// ListSmallChunks returns the small chunks inside the large chunk that encloses each given chunk of one
// vault, in the order of the given chunks. A chunk the vault does not hold has none.
func (q *Search) ListSmallChunks(ctx context.Context, vaultID domain.VaultID, ids []domain.ChunkID) (_ []source.SmallChunk, err error) {
	if len(ids) == 0 {
		return nil, nil
	}
	vault, err := getVaultRow(ctx, q.db, vaultID)
	if errors.Is(err, errNoVault) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows := make([]int64, len(ids))
	for i, id := range ids {
		if rows[i], err = ParseRow(id); err != nil {
			return nil, err
		}
	}
	found, err := q.db.QueryContext(ctx, smallChunksSQL, formatInt64s(rows), vault)
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, found.Close()) }()

	var out []source.SmallChunk
	for found.Next() {
		var given, small int64
		if err := found.Scan(&given, &small); err != nil {
			return nil, err
		}
		out = append(out, source.SmallChunk{Chunk: FormatID(given), Small: FormatID(small)})
	}
	return out, found.Err()
}
