/** What a deck tab and a stencil tab say: what they are called, and what is wrong. */
// Stryker disable StringLiteral: interface copy
import { many } from '@numen/ui'
import { getFileName } from '@numen/plugin-sdk'

export const WORDS = {
  deck: 'Deck',
  stencil: 'Stencil',
  /**
   * The field a stencil is made carrying, which is the one every card's heading
   * is read from. A stencil declaring no field is unsound by the format.
   */
  newField: 'Field 1',
  /** The two the file puts to the person, and the two ways out. */
  stale: 'Saving paused: the file changed on disk.',
  gone: 'Saving paused: the file was removed from the vault. Your text is kept.',
  makeAgain: 'recreate it',
  keep: 'Keep my version',
  take: 'Use disk version',
  /** What stopped the whole file being read, said above what was read. */
  notRead: 'File unreadable.',
  notSaved: 'File not saved.',
  /** The vault answered nothing at all, and what is on screen is still here. */
  unreachable: 'Vault unreachable.',
  notADeck: 'Not a deck.',
  notAStencil: 'Not a stencil.',
  /** A deck past the size one is read at, with the bound it is past. */
  tooLarge: (linked: number) => `Deck larger than ${linked} bytes; not loaded.`,
  /** The line at the top of a deck, which says which preset schedules it. */
  scheduledBy: 'Scheduled by',
  defaults: 'Defaults',
  unnamed: (path: string) => getFileName(path),
  /** A deck the vault would not put on the preset chosen. */
  notScheduled: 'Preset not applied to this deck.',
  /** A deck the file has moved past since the window read it. */
  notScheduledChanged: 'Preset not applied: the deck changed on disk.',
  /** What is wrong with the file itself, standing against no card and no face. */
  problems: 'File problems',
  /** What a mark on a card, a face or a field is announced as. */
  wrong: 'Problem',
  /** How far a field's new name reached, said once it has. */
  renamed: (cards: number, decks: number) =>
    `Field renamed in ${many(cards, 'card')} across ${many(decks, 'deck')}.`,
  /** A rename asked against a stencil the file has since moved past. */
  notRenamed: 'Field not renamed: the file changed on disk.',
  /** The decks the new name did not reach, each keeping the heading it had. */
  notWritten: (paths: readonly string[]) => `Old heading kept in: ${paths.join(', ')}.`,
}
