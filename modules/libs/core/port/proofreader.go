package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Proofreader is asked about the lines of a request and says what it would put
// right. The core asks for one and does not know whether a model on this machine
// or a service answered.
//
// An installation that configured none has no proofreader, and a recognition is
// used exactly as it was read.
type Proofreader interface {
	// GetName is what made a correction, recorded beside it.
	GetName() string

	// Proofread answers a request with the corrections the proofreader stands
	// behind. A group nothing came back about is left as it was read.
	Proofread(ctx context.Context, request domain.ProofreadRequest) (domain.ProofreadAnswer, error)
}

// A ProofreadQueue is a proofreader that takes a request away and answers about
// it later. The answer outlives the run that asked for it, so a request left
// before the application closed is collected when it opens.
type ProofreadQueue interface {
	Proofreader

	// Submit hands the request over and answers with the name what comes back is
	// collected under.
	Submit(ctx context.Context, request domain.ProofreadRequest) (string, error)

	// GetAnswer is what came back about the request left under a name, and
	// whether it is there yet. A name the service will never answer about is an
	// error.
	GetAnswer(ctx context.Context, name string) (answer domain.ProofreadAnswer, ready bool, err error)
}
