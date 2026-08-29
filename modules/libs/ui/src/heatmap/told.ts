/**
 * What the grid says about one day, in words.
 *
 * The words are the window's and not this library's: it holds no name for a
 * month, an answer or a day, so a person reads their own language wherever the
 * grid is drawn.
 */

/** What each line of a day's account is called. */
export interface Words {
  /** The day itself, said however the window says a date. */
  readonly names: (day: string) => string
  /** "answered", after a number: what was done on a day behind. */
  readonly answered: string
  /** What a day nobody answered on says. */
  readonly nothing: string
  /** "to come", after a number: what falls on a day still ahead. */
  readonly toCome: string
  /** The four, as they are named on the buttons a person presses. */
  readonly again: string
  readonly hard: string
  readonly good: string
  readonly easy: string
  /** "recalled", after a share: how much of what was learned came back. */
  readonly recalled: string
}
