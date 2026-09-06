/**
 * The words the pill saying how many cards are due is drawn with. No DOM, no
 * measurement, no clock.
 */

/** The words the pill is drawn with, declared once. */
export interface DueWords {
  /** The figure and what it counts, as one reading. */
  readonly counted: (due: number) => string
  /** What is said while the figure is still being worked out. */
  readonly counting: string
}

export const DUE_WORDS: DueWords = {
  counted: (due) => `${due} to review`,
  counting: 'still being counted',
}
