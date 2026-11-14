package cards

import (
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

func newCardIDs(entropy ports.Entropy) func() format.ID {
	return func() format.ID { return format.CreateID(entropy.Fill) }
}
