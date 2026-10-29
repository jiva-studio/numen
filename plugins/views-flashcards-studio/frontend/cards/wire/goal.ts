/**
 * What a window calls the goal of a preset, and what it sends back.
 *
 * Both windows say the goal in the same three words: the editor sets it and the
 * flashcards window reads it off a preset it did not write. Two tables would
 * agree until one of them was given the next goal.
 */
import { Goal as Goals } from '../../api/types'

import { getNames } from '../../api/names'

/** Which value the one control of a preset steers. */
export type Goal = 'minutes' | 'retention' | 'date'

/**
 * The goal as the window names it, and null for a preset naming none: the
 * default belongs to the caller. Keyed by the schema, so a goal added
 * to it requires a word here before this compiles.
 */
export const goalOf: Readonly<Record<Goals, Goal | null>> = {
  [Goals.UNSPECIFIED]: null,
  [Goals.MINUTES_A_DAY]: 'minutes',
  [Goals.RETENTION]: 'retention',
  [Goals.BY_DATE]: 'date',
}

/** The goal as the schema names it, read off the words above. */
export const goalNames: Readonly<Record<Goal, Goals>> = getNames<Goal, Goals>(goalOf)
