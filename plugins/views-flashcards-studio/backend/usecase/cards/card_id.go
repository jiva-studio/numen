package cards

import (
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
)

// Entropy fills a card id with random bytes.
type Entropy interface {
	Fill(b []byte)
}

func newCardIDs(entropy Entropy) func() format.ID {
	return func() format.ID { return format.CreateID(entropy.Fill) }
}
