package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Proofreader puts numbered lines of text right. The core asks for one and does
// not know which plugin answers, which model it asks, or how it cuts the text
// into requests.
//
// An installation that configured none has no proofreader, and a recognition is
// used exactly as it was read. A proofreader with nothing routed to answer for
// it fails with ErrUnavailable.
type Proofreader interface {
	// GetName is what made a correction, recorded beside it.
	GetName() string

	// ProofreadScan puts right the lines of printed pages. A correction stands
	// no further than maxEditDistance from its line, as a share of the longer
	// of the two; zero leaves the limit to the proofreader.
	ProofreadScan(ctx context.Context, pages []domain.ProofreadPage, maxEditDistance float64) (domain.ScanProofreading, error)

	// ProofreadSpeech puts right the lines of a transcript that fall in the
	// window. The lines are the whole transcript, so a proofreader knows what
	// the recording holds.
	ProofreadSpeech(ctx context.Context, lines []domain.ProofreadLine, window domain.SpeechWindow) (domain.SpeechProofreading, error)
}
