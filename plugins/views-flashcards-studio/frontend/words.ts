/** What a deck tab and a card model tab say: what they are called, and what is wrong. */
// Stryker disable StringLiteral: interface copy
import { many } from '@numen/ui'
import { getFileName } from '@numen/plugin-sdk'

export const WORDS = {
  deck: 'Deck',
  cardModel: 'Card model',
  /**
   * The field a card model is made carrying, which is the one every card's heading
   * is read from. A card model declaring no field is unsound by the format.
   */
  newField: 'Field 1',
  /** The two messages for a file that changed or was removed, and the two ways out. */
  stale: 'Saving paused: the file changed on disk.',
  gone: 'Saving paused: the file was removed from the vault. Your text is kept.',
  makeAgain: 'recreate it',
  keep: 'Keep my version',
  take: 'Use disk version',
  /** What prevented the whole file from being read, shown above the partial contents. */
  notRead: 'File unreadable.',
  notSaved: 'File not saved.',
  /** The vault answered nothing at all, and what is on screen is still here. */
  unreachable: 'Vault unreachable.',
  notADeck: 'Not a deck.',
  notACardModel: 'Not a card model.',
  /** A deck past the size one is read at, with the bound it is past. */
  tooLarge: (linked: number) => `Deck larger than ${linked} bytes; not loaded.`,
  /** The line at the top of a deck that names the preset scheduling it. */
  scheduledBy: 'Scheduled by',
  defaults: 'Defaults',
  unnamed: (path: string) => getFileName(path),
  /** A deck the vault would not put on the preset chosen. */
  notScheduled: 'Preset not applied to this deck.',
  /** A deck the file has moved past since the window read it. */
  notScheduledChanged: 'Preset not applied: the deck changed on disk.',
  /** What is wrong with the file itself, standing against no card and no template. */
  problems: 'File problems',
  /** What a mark on a card, a template or a field is announced as. */
  wrong: 'Problem',
  /** How many cards and decks a field rename reached, shown once the rename completes. */
  renamed: (cards: number, decks: number) =>
    `Field renamed in ${many(cards, 'card')} across ${many(decks, 'deck')}.`,
  /** A rename requested against a card model that changed on disk since it was read. */
  notRenamed: 'Field not renamed: the file changed on disk.',
  /** The decks the new name did not reach, each keeping the heading it had. */
  notWritten: (paths: readonly string[]) => `Old heading kept in: ${paths.join(', ')}.`,
}
