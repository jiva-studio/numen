package cards

import (
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
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
// write changed up to date in the index. Entropy fills the card ids given to cards that carry none,
// and the identifiers of the notes written.
type Config struct {
	Readers    ports.VaultReaders
	Writers    ports.VaultWriters
	Notes      TypeQueries
	CardModels CardModelQueries
	Links      ports.LinkQueries
	Index      edit.IndexUpdater
	Now        ports.Clock
	Entropy    ports.Entropy
}

// NewScenarios builds the scenarios on the notes of a card. Its rename takes a vault of any size.
func NewScenarios(cfg Config) Scenarios {
	return Scenarios{
		Read:        NewRead(cfg.Readers, cfg.Links),
		List:        NewList(cfg.Readers, cfg.CardModels),
		Write:       NewWrite(cfg.Readers, cfg.Writers, cfg.Links, cfg.Index, cfg.Now, cfg.Entropy),
		Create:      NewCreate(cfg.Writers, cfg.Index, cfg.Now, cfg.Entropy),
		RenameField: NewRenameField(cfg, UnboundedDecks),
	}
}

// NewAgentRename builds the rename an agent calls, which refuses a vault of many decks.
func NewAgentRename(cfg Config) RenameField {
	return NewRenameField(cfg, MaxDecksPerRename)
}

// NewDeckWriter returns the writer of a deck over cfg, with index as what brings the index up to
// date after a write.
func (cfg Config) NewDeckWriter(index edit.IndexUpdater) ports.DeckWriter {
	return NewWrite(cfg.Readers, cfg.Writers, cfg.Links, index, cfg.Now, cfg.Entropy)
}
