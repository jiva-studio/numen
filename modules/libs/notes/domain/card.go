package domain

// CardID is what a card is, for as long as it exists. It is minted once, and
// it travels with the card: the same card moved to another deck or another
// vault is the same identifier, and nothing about it is recomputed. A card's
// heading writes it behind a caret, where it is called the card's mark.
type CardID string
