package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// Proofreader is asked about the lines of a run of batches and says what it
// would put right. The core asks for one and does not know whether a model on
// this machine or a service answered.
//
// An installation that configured none has no proofreader, and a reading is
// used exactly as it was read.
type Proofreader interface {
	// GetName is what made a correction, recorded beside it.
	GetName() string

	// Proofread hands over the batches and answers with what came back about
	// each, by the batch it is about. Whether a reply is an answer is decided
	// above this; a batch nothing came back about is a batch left as it was
	// read.
	Proofread(ctx context.Context, batches []proofread.Batch) (map[int]string, error)
}

// A ProofreadQueue is a proofreader that takes a run of batches away and
// answers about them later. The answer outlives the run that asked for it, so a
// batch left before the application closed is collected when it opens.
type ProofreadQueue interface {
	Proofreader

	// Leave hands the batches over and answers with the name what comes back is
	// collected under.
	Leave(ctx context.Context, batches []proofread.Batch) (string, error)

	// Collect is what came back about the batches left under a name, and
	// whether it is there yet. A name the service will never answer about is an
	// error.
	Collect(ctx context.Context, name string) (replies map[int]string, ready bool, err error)
}
