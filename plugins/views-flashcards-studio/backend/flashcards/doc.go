// Package flashcards holds the use cases for running a vault's cards: reading
// the decks and card models, assigning card ids and writing the answers.
//
// What an answer means and where a history leaves a card is defined by the
// core's review package. This package connects it to a vault: decks are read
// from files, a deck containing a card without a card id is written so that the
// card has one, and the answers are appended to the vault's own folder.
package flashcards
