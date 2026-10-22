package cards

// Scenarios is everything that acts on the two notes a flashcard is made of.
// Both the window and the tools an agent calls are served the same set, so what
// one of them refuses the other refuses.
type Scenarios struct {
	Read        Read
	List        List
	Write       Write
	Create      Create
	RenameField RenameField
	// BoundedRenameField refuses a vault holding more than MaxDecksPerRename decks.
	BoundedRenameField RenameField
}
