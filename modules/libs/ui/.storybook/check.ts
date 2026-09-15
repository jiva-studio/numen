/**
 * What a story is judged by once it has been played, wherever it was drawn.
 *
 * Every package that runs stories runs this after each of them, so a stop the
 * components refuse is refused in the windows too, and there is one walk to
 * fix when the rule changes.
 */
import type { Preview } from '@storybook/vue3-vite'
import { walk } from './keyboard'
import { faults } from './reach'

/**
 * The story a corpus proves its own walk against, and what the walk must find
 * there. A walk that finds nothing passes everything, so each corpus names one
 * story of its own crowded enough that finding nothing in it means the walk has
 * stopped working — and a floor named in another package's story proves nothing
 * about this one.
 */
export interface Proof {
  /** The story's id, as Storybook addresses it. */
  readonly story: string
  /** The fewest stops the keyboard finds there. */
  readonly stops: number
}

/** What the page not coming to rest is said with, where it may be the cause. */
const RESTLESS =
  'and the page had not come to rest, so these may be what it was drawn as on the way'

/**
 * What a story says of the walk: `false` to leave it unwalked, or the way out
 * of something that answers Tab by keeping it.
 */
export type Reach = false | { readonly keeps: string }

/** The check, wired to a corpus by the story it is proved against. */
export const reachCheck =
  (proof: Proof): NonNullable<Preview['afterEach']> =>
  async (context) => {
    const said = context.parameters['reach'] as Reach | undefined
    if (said === false) return

    const found = await walk()

    if (context.id === proof.story && found.stops.length < proof.stops)
      throw new Error(
        `the keyboard walk found ${found.stops.length} stops in ${proof.story}, where it must find ${proof.stops}`,
      )

    const wrong = faults(found, Boolean(said && said.keeps))
    if (wrong.length)
      throw new Error(`${context.id}: ${wrong.join('; ')}${found.settled ? '' : ` — ${RESTLESS}`}`)
  }
