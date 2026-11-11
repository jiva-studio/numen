package cards

import (
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/domain/format"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/internal/adapter/clock"
)

func createCardID() format.ID {
	return format.CreateID(clock.ReadEntropy)
}
