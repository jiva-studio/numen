package webui

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// A transcript a model heard is put right by a second model, asked for as the
// corrections that stand beside the recording. The work carries on behind the
// answer, and the answer says what those corrections now are.

// Proofreader puts the transcript of a recording right, for a person who asked
// for it.
type Proofreader interface {
	// ProofreaderReady says whether this installation has anything to put a
	// transcript right with.
	ProofreaderReady() bool
	// Proofread puts one transcript right and says what came of asking. It
	// answers before the work is over.
	Proofread(ctx context.Context, v domain.Vault, path string) (source.PutRightResult, error)
}

// errNoProofreading is a build, or an installation, with nothing to put a
// transcript right with.
var errNoProofreading = errors.New("this build cannot proofread a transcript")
