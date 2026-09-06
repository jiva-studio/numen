package flashcards_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/flashcards/review"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// closing is the vault's files with the request cancelled at the first one
// read, which is a person shutting the window while its decks are being read.
type closing struct {
	inner port.VaultReaders
	at    context.CancelFunc
}

func (r closing) Open(v domain.Vault) (port.VaultReader, error) {
	one, err := r.inner.Open(v)
	if err != nil {
		return nil, err
	}
	return closingReader{VaultReader: one, at: r.at}, nil
}

type closingReader struct {
	port.VaultReader
	at context.CancelFunc
}

func (r closingReader) Read(ctx context.Context, path string) ([]byte, error) {
	r.at()
	return r.VaultReader.Read(ctx, path)
}

// A deck that could not be read contributes no card face, and a request that
// was cancelled read none of them. A curve over no cards is not the answer to
// what the preset holds.
func TestACurveRefusesARequestThatIsGone(t *testing.T) {
	t.Parallel()
	s := answering(t, 30)
	ctx, cancel := context.WithCancel(t.Context())

	curves := s.curves(noon)
	curves.CardFaces.Readers = closing{inner: curves.CardFaces.Readers, at: cancel}

	p := review.Preset{Goal: review.GoalMinutes, MinutesADay: 20, NewADay: 8, ReviewsADay: 45}
	got, err := curves.Execute(ctx, s.vault, "Sanskrit.md", p)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("a curve drawn after the request was cancelled gave %+v, %v", got, err)
	}
}
