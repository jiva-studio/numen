/** What a deck tab and a stencil tab say: what they are called, and what is wrong. */
import { many } from '@numen/ui'
import { fileOf } from '../../shared/paths'

export const WORDS = {
  deck: 'Deck',
  stencil: 'Stencil',
  /**
   * The field a stencil is made carrying, which is the one every card's heading
   * is read from. A stencil declaring no field is unsound by the format.
   */
  newField: 'Field 1',
  /** The two the file puts to the person, and the two ways out. */
  stale: 'This file changed on disk, so it stopped saving.',
  gone: 'This file is no longer in the vault, so saving stopped. What is here is still yours.',
  makeAgain: 'make it again',
  keep: 'Keep mine',
  take: "Take the file's",
  /** What the whole file is refused for, said above what was read. */
  refused: 'This file could not be read.',
  /** What a write of the whole file is refused for, said the same way. */
  notSaved: 'This file could not be written.',
  /** The vault answered nothing at all, and what is on screen is still here. */
  unreachable: 'The vault could not be reached.',
  notADeck: 'That note is not a deck.',
  notAStencil: 'That note is not a stencil.',
  /** A deck past the size one is read at, with the bound it is past. */
  tooLarge: (bound: number) => `This deck is over ${bound} bytes, so none of it was read.`,
  /** The line at the top of a deck, which says which preset schedules it. */
  scheduledBy: 'Scheduled by',
  /** The choice a deck naming no preset stands at. */
  defaults: 'The defaults',
  /** A preset the note carries no name for, drawn by the file it stands in. */
  unnamed: (path: string) => fileOf(path),
  /** A deck the vault would not put on the preset chosen. */
  notScheduled: 'This deck was not put on that preset.',
  /** A deck the file has moved past since the window read it. */
  notScheduledChanged: 'This deck changed on disk, so it was not put on that preset.',
  /** What is wrong with the file itself, standing against no card and no face. */
  problems: 'What is wrong with this file',
  /** What a mark on a card, a face or a field is announced as. */
  wrong: 'Wrong here',
  /** How far a field's new name reached, said once it has. */
  renamed: (cards: number, decks: number) =>
    `The field was renamed in ${many(cards, 'card')}, over ${many(decks, 'deck')}.`,
  /** A rename asked against a stencil the file has since moved past. */
  notRenamed: 'This file changed on disk, so the field was not renamed.',
  /** The decks the new name did not reach, each keeping the heading it had. */
  notWritten: (paths: readonly string[]) =>
    `These decks keep the old heading: ${paths.join(', ')}.`,
}
