package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// IndexProgress is how far reading a vault for meaning has got.
//
// It is asked for while the work is still running, which is the only time the
// answer is interesting: cutting a vault finishes long before embedding it does,
// so the two counts together are what says how much is left. A vault nothing has
// cut yet holds nothing and owes nothing.
type IndexProgress interface {
	// Progress is how many of the vault's chunks can carry a vector, and how
	// many of those carry one for the model named. The population is the same
	// one an answer about what owes a vector is drawn from, so the two agree and
	// the count reaches the total.
	//
	// A chunk with no vector is searchable by its words and not by its meaning,
	// and a vault spends most of its life that way.
	Progress(ctx context.Context, vaultID domain.VaultID, model string) (held, embedded int64, err error)
}
