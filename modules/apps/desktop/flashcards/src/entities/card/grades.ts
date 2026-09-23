/** How well a card came back, in this window's words and in the schema's. */
import { Rating } from '@numen/protocol'
import { namesOf } from '@numen/wire'

/** How well a card came back. A person says which of the four. */
export type Grade = 'again' | 'hard' | 'good' | 'easy'

/** The four, in the order they are offered and answered by number. */
export const grades: readonly Grade[] = ['again', 'hard', 'good', 'easy']

/** What each of them is called on the button that says it. */
export const called: Readonly<Record<Grade, string>> = {
  again: 'Again',
  hard: 'Hard',
  good: 'Good',
  easy: 'Easy',
}

/**
 * What each of them is called in this window's own words. Keyed by the schema,
 * so a rating added to it has to be given a word here before this compiles.
 */
const graded: Readonly<Record<Rating, Grade | null>> = {
  [Rating.UNSPECIFIED]: null,
  [Rating.AGAIN]: 'again',
  [Rating.HARD]: 'hard',
  [Rating.GOOD]: 'good',
  [Rating.EASY]: 'easy',
}

/** What the schema calls each of them, read off the words above. */
export const rated: Readonly<Record<Grade, Rating>> = namesOf<Grade, Rating>(graded)
