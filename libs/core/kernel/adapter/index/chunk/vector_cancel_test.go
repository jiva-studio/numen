package chunk_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index/chunk"
)

// The vec0 table prepares its own statements on the connection's first write to
// it. Canceling the context interrupts the connection, and a prepare that is
// interrupted leaves vec0 with a nil statement that it then clears, which
// crashes the process. A save canceled at any moment must return an error and
// leave the process alive.
func TestSavingVectorsCanceledAtAnyMomentDoesNotCrash(t *testing.T) {
	for i := range 40 {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			s := newStore(t)
			var cut []chunk.Chunk
			for j := range 40 {
				cut = append(cut, chunk.Chunk{Start: j * 5, Length: 5, Text: fmt.Sprintf("chunk %d", j)})
			}
			s.cut(repoVaultA, "book", "a.epub", cut)
			var vectors []chunk.Vector
			for _, row := range s.rowsOf(repoVaultA, "a.epub") {
				vectors = append(vectors, s.vectorFor(row, "m", 1))
			}

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			go func() {
				time.Sleep(time.Duration(i) * 40 * time.Microsecond)
				cancel()
			}()

			// Finishing, being canceled, or finding the transaction rolled back by the cancellation are all fine; the process surviving is the point.
			if err := s.db.Chunks().SaveVectorsInBatches(ctx, vectors, 7); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, sql.ErrTxDone) {
				t.Error(err)
			}
		})
	}
}
