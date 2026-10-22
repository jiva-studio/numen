package editor

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/recognition"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/transcription"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
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
	Proofread(ctx context.Context, v domain.Vault, path string) (transcription.ProofreadResult, error)
}

// RecognitionProofreader puts a document's recognition right, for a person who asked
// for it.
type RecognitionProofreader interface {
	// ProofreaderReady says whether this installation has anything to put a
	// recognition right with.
	ProofreaderReady() bool
	// Proofread puts one recognition right and says what came of asking. It
	// answers before the work is over.
	Proofread(ctx context.Context, v domain.Vault, path string) (recognition.ProofreadResult, error)
}

// errNoProofreading is an installation the settings name nothing to put a
// transcript or recognition right with.
var errNoProofreading = errors.New("this installation has no proofreader")

// errNotRead is a document not processed yet.
var errNotRead = errors.New("document not processed")
