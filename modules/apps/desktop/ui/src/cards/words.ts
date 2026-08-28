/** What a deck tab and a stencil tab say: what they are called, and what is wrong. */
export const WORDS = {
  deck: 'Deck',
  stencil: 'Stencil',
  /** What a file is called before a name has been typed over it. */
  newDeck: 'New deck',
  newStencil: 'New stencil',
  /**
   * The field a stencil is made carrying, which is the one its cards are named
   * by. A stencil declaring no field is unsound by the format.
   */
  newField: 'Field 1',
  /** The two the file puts to the person, and the two ways out. */
  overtaken: 'This file changed on disk, so it stopped saving.',
  gone: 'This file is no longer in the vault, so saving stopped. What is here is still yours.',
  makeAgain: 'make it again',
  keep: 'Keep mine',
  take: "Take the file's",
  /** What the whole file is refused for, said above what was read. */
  refused: 'This file could not be read.',
  notADeck: 'That note is not a deck.',
  notAStencil: 'That note is not a stencil.',
  /** A deck past the size one is read at, with the bound it is past. */
  tooLarge: (bound: number) => `This deck is over ${bound} bytes, so none of it was read.`,
  /** What is wrong with the file itself, standing against no card and no face. */
  problems: 'What is wrong with this file',
  /** What a mark on a card, a face or a field is announced as. */
  wrong: 'Wrong here',
  /** How far a field's new name reached, said once it has. */
  renamed: (cards: number, decks: number) =>
    `The field was renamed in ${cards} ${cards === 1 ? 'card' : 'cards'},` +
    ` over ${decks} ${decks === 1 ? 'deck' : 'decks'}.`,
  /** The decks the new name did not reach, each keeping the heading it had. */
  notWritten: (paths: readonly string[]) =>
    `These decks keep the old heading: ${paths.join(', ')}.`,
}
