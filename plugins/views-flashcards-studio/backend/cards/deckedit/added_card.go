package deckedit

// AddedCard is a deck after a card was added to it, and the card id the card is addressed by.
type AddedCard struct {
	EditResult
	CardID string
}
