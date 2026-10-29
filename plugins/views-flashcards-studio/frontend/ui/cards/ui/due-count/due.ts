/**
 * The words the pill saying how many cards are due is drawn with. No DOM, no
 * measurement, no clock.
 */

/** The words the pill is drawn with, declared once. */
export interface DueWords {
  /** The figure and what it counts, as one reading. */
  readonly formatDue: (due: number) => string
  /** Text shown while the figure is still being computed. */
  readonly counting: string
}

export const DUE_WORDS: DueWords = {
  formatDue: (due) => `${due} to review`,
  counting: 'still being counted',
}
