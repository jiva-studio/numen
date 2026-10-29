/**
 * The text the grid displays for one day.
 *
 * The text is supplied by the window and not by this library: it contains no
 * name for a month, an answer or a day, so the grid is displayed in the
 * user's language wherever it is used.
 */
// Stryker disable StringLiteral: interface copy

/** What each line of a day's account is called. */
export interface Words {
  /** The day itself, formatted as the window formats a date. */
  readonly names: (day: string) => string
  /** "answered", after a number: what was done on a day behind. */
  readonly reviewed: string
  /** The text for a day with no answers. */
  readonly nothing: string
  /** "to come", after a number: what falls on a day still ahead. */
  readonly toCome: string
  /** The four ratings, as named on the answer buttons. */
  readonly again: string
  readonly hard: string
  readonly good: string
  readonly easy: string
  /** What follows a share: what the cards it is a share of are. */
  readonly recalled: string
}
