// Package cards contains the use cases that operate on the two notes a
// flashcard is made of: the card model, which defines what a card contains, and
// the deck, which contains the cards.
//
// The format package defines what a deck and a card model are made of. This
// package connects them to a vault: a file's metadata is checked before it is
// opened, a write is checked against what the caller read, and a field renamed
// in a card model is renamed in every deck that card model cuts.
package cards
