package cards

// Scenarios is everything that acts on the two notes a flashcard is made of.
// The window and the tools an agent calls use the same set, so both reject the
// same inputs.
type Scenarios struct {
	Read        Read
	List        List
	Write       Write
	Create      Create
	RenameField RenameField
	// BoundedRenameField rejects a vault holding more than MaxDecksPerRename decks.
	BoundedRenameField RenameField
}
