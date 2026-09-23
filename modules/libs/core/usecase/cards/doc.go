// Package cards holds the scenarios that act on the two notes a flashcard is
// made of: the stencil that says what a card has, and the deck that holds the
// cards.
//
// What a deck and a stencil are made of is the format package. This is where
// they meet a vault: a file is looked at before it is opened, a write is held
// to what the caller read, and a field renamed in a stencil reaches every deck
// that stencil cuts.
package cards
