package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/proofread"
)

// Proofreader is asked about the printed lines of a run of pages and says what
// it would put right. The core asks for one and does not know whether a model
// on this machine or a service answered.
//
// An installation that configured none has no proofreader, and a reading is
// used exactly as it was read.
type Proofreader interface {
	// Name is what made a correction, recorded beside it.
	Name() string

	// Read hands over the pages and answers with what came back about each, by
	// the page it is about. Whether a reply is an answer is decided above this;
	// a page nothing came back about is a page left as it was read.
	Read(ctx context.Context, pages []proofread.Page) (map[int]string, error)
}

// A ProofreadQueue is a proofreader that takes a run of pages away and answers
// about them later. The answer outlives the run that asked for it, so a batch
// left before the application closed is collected when it opens.
type ProofreadQueue interface {
	Proofreader

	// Leave hands the pages over and answers with the name what comes back is
	// collected under.
	Leave(ctx context.Context, pages []proofread.Page) (string, error)

	// Collect is what came back about the pages left under a name, and whether
	// it is there yet. A name the service will never answer about is an error.
	Collect(ctx context.Context, name string) (replies map[int]string, ready bool, err error)
}
