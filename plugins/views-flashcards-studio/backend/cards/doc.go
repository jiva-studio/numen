// Package cards holds the scenarios that act on the two notes a flashcard is
// made of: the card model that says what a card has, and the deck that holds the
// cards.
//
// What a deck and a card model are made of is the format package. This is where
// they meet a vault: a file is looked at before it is opened, a write is held
// to what the caller read, and a field renamed in a card model reaches every deck
// that card model cuts.
package cards
