/**
 * What a window calls the goal of a preset, and what it sends back.
 *
 * Both windows say the goal in the same three words: the editor sets it and the
 * flashcards window reads it off a preset it did not write. Two tables would
 * agree until one of them was given the next goal.
 */
import { Goal as Goals } from '@numen/protocol'

import { namesOf } from './naming'

/** Which value the one control of a preset steers. */
export type Goal = 'minutes' | 'retention' | 'date'

/**
 * The goal in a window's own words, and nothing for a preset naming none: what
 * the default is belongs to whoever asked. Keyed by the schema, so a goal added
 * to it has to be given a word here before this compiles.
 */
export const goalOf: Readonly<Record<Goals, Goal | null>> = {
  [Goals.UNSPECIFIED]: null,
  [Goals.MINUTES_A_DAY]: 'minutes',
  [Goals.RETENTION]: 'retention',
  [Goals.BY_DATE]: 'date',
}

/** The goal as the schema names it, read off the words above. */
export const goalNames: Readonly<Record<Goal, Goals>> = namesOf<Goal, Goals>(goalOf)
