package cards

import (
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/note"
)

// Scenarios is what acts on the two notes a flashcard is made of. The window and the tools an agent
// calls use the same set, so both reject the same inputs; an agent's rename is bounded and is
// built apart.
type Scenarios struct {
	Read        Read
	List        List
	Write       Write
	Create      Create
	RenameField RenameField
}

// Config is what the scenarios on the notes of a card are built over. Index brings the notes a
// write changed up to date in the index.
type Config struct {
	Readers    ports.VaultReaders
	Writers    ports.VaultWriters
	Notes      TypeQueries
	CardModels CardModelQueries
	Links      ports.LinkQueries
	Index      note.IndexUpdater
	Now        ports.Clock
}

// NewScenarios builds the scenarios on the notes of a card. Its rename takes a vault of any size.
func NewScenarios(cfg Config) Scenarios {
	return Scenarios{
		Read:        NewRead(cfg.Readers, cfg.Links),
		List:        NewList(cfg.Readers, cfg.CardModels),
		Write:       NewWrite(cfg.Readers, cfg.Writers, cfg.Links, cfg.Index, cfg.Now),
		Create:      NewCreate(cfg.Writers, cfg.Index, cfg.Now),
		RenameField: NewRenameField(cfg, UnboundedDecks),
	}
}

// NewAgentRename builds the rename an agent calls, which refuses a vault of many decks.
func NewAgentRename(cfg Config) RenameField {
	return NewRenameField(cfg, MaxDecksPerRename)
}
