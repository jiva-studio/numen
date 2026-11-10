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
  readonly names: (day: string) => string
  readonly reviewed: string
  readonly nothing: string
  readonly toCome: string
  readonly again: string
  readonly hard: string
  readonly good: string
  readonly easy: string
  readonly recalled: string
}
