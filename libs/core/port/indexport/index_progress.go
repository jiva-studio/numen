package indexport

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// IndexProgress is how far indexing a vault for meaning has got.
//
// It is queried while the work is still running, which is the only time the
// result is useful: chunking a vault finishes long before embedding does, so
// the two counts together show how much remains. A vault that has not been
// chunked has no chunks and none to embed.
type IndexProgress interface {
	// GetProgress returns how many of the vault's chunks can have a vector, and
	// how many of those have one for the named model. The population is the same
	// one the query for chunks that need a vector draws from, so the two agree
	// and the embedded count reaches the total.
	//
	// A chunk with no vector is searchable by its words and not by its meaning,
	// and a vault spends most of its life that way.
	GetProgress(ctx context.Context, vaultID domain.VaultID, model string) (held, embedded int64, err error)
}
