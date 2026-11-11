package chunk

import (
	"context"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/sqlwrite"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// ListStoredVectors returns the vectors kept under the recipe for the chunks, in the order the
// database yields them.
func (q *Search) ListStoredVectors(ctx context.Context, query source.StoredVectorQuery) (_ []source.StoredVector, err error) {
	if len(query.Chunks) == 0 {
		return nil, nil
	}
	rows := make([]int64, len(query.Chunks))
	for i, id := range query.Chunks {
		if rows[i], err = Row(id); err != nil {
			return nil, err
		}
	}
	found, err := q.db.QueryContext(ctx, rerankSQL, formatInt64s(rows), query.Recipe, formatKinds(query.Kinds))
	if err != nil {
		return nil, err
	}
	defer func() { err = sqlwrite.JoinClose(err, found.Close()) }()

	var out []source.StoredVector
	for found.Next() {
		var row int64
		var stored []byte
		if err := found.Scan(&row, &stored); err != nil {
			return nil, err
		}
		out = append(out, source.StoredVector{Chunk: ID(row), Vector: stored})
	}
	return out, found.Err()
}

func formatInt64s(nums []int64) string {
	if len(nums) == 0 {
		return "[]"
	}
	// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated values are 13 and 3: a larger capacity hint; the text is the same at any capacity"
	b := make([]byte, 0, len(nums)*12+2)
	b = append(b, '[')
	for i, n := range nums {
		if i > 0 {
			b = append(b, ',')
		}
		b = strconv.AppendInt(b, n, 10)
	}
	b = append(b, ']')
	return string(b)
}

func formatKinds(selected []domain.SourceKind) string {
	if len(selected) == 0 {
		return "[]"
	}
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="the mutated values are 1 and 3: the text needs 1 byte less than the hint, so a hint one smaller or larger yields the same text without growing"
	size := 2
	for _, k := range selected {
		// gomutants:disable-next-line INTEGER_INCREMENT reason="the mutated value is 4: a larger capacity hint; the text is the same at any capacity"
		size += len(k) + 3
	}
	b := make([]byte, 0, size)
	b = append(b, '[')
	for i, k := range selected {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, '"')
		b = append(b, k...)
		b = append(b, '"')
	}
	b = append(b, ']')
	return string(b)
}
