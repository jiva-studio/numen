/**
 * What a line of activity is, as plain values. No DOM, no clock, no
 * measurement.
 *
 * Activity is work that runs whether or not anyone asked: a count that is
 * climbing, and a name for what is climbing it. Whoever renders this decides
 * what the words are, so nothing here knows what is being counted.
 */

import { clock } from '@/shared/lib/duration'

/**
 * Where a piece of work has got to.
 *
 * `resting` has words and no work: something worth knowing that nothing is
 * going to change by itself. It is drawn without the marks of progress.
 */
export type ActivityState = 'quiet' | 'working' | 'resting' | 'failed'

/**
 * How a line reads.
 *
 * Given by whoever draws the line, where the state above is worked out from the
 * count. Alarm is what a line that failed is drawn with.
 */
export type Tone = 'plain' | 'caution' | 'alarm'

/** A count of things done out of things to do. */
export interface Tally {
  readonly done: number
  readonly total: number
}

/**
 * What a count counts.
 *
 * Bytes are read out in the sizes a person reads them in and seconds on a
 * clock. Everything else is counted one by one.
 */
export type TallyUnit = 'things' | 'bytes' | 'seconds'

/** What a line of activity draws. */
export interface ActivityDescriptor {
  readonly state: ActivityState
  /** Whether a bar is drawn, and how full. Absent when the share is unknown. */
  readonly share?: number
  /** Whether the count is worth showing beside the words. */
  readonly hasCounts: boolean
}

/**
 * The share of a tally that is done.
 *
 * A total of nothing has no share: zero out of zero is finished and empty at
 * once, and a bar cannot draw both. A count beyond its total reads as full,
 * because a tally that grows while it is being worked through overtakes itself
 * for as long as it takes the total to catch up.
 */
export const shareOf = (tally: Tally): number | undefined => {
  if (tally.total <= 0) return undefined
  if (tally.done >= tally.total) return 1
  if (tally.done <= 0) return 0
  return tally.done / tally.total
}

/**
 * What to draw for a piece of work.
 *
 * Nothing to say is quiet, and a line with nothing to say draws nothing.
 * Failure outranks every other state: a line that is both failing and counting
 * says it is failing. Quiet draws nothing — which is not
 * the same as finished. Everything else is resting or working, and a count is
 * what makes the difference visible.
 */
export const activity = (input: {
  readonly text: string
  readonly hasFailed?: boolean
  readonly isWorking?: boolean
  readonly tally?: Tally
}): ActivityDescriptor => {
  if (!input.text) return { state: 'quiet', hasCounts: false }
  if (input.hasFailed) return { state: 'failed', hasCounts: false }

  // Work is claimed, not assumed. Words alone say something is so, and a caller
  // that means "this is happening now" says that too.
  const share = input.tally ? shareOf(input.tally) : undefined
  if (!input.isWorking && share === undefined) return { state: 'resting', hasCounts: false }

  return share === undefined
    ? { state: 'working', hasCounts: false }
    : { state: 'working', share, hasCounts: true }
}

/**
 * A share as a percentage, for reading beside the count.
 *
 * Rounded down, so that nothing says a hundred per cent until it is finished.
 * Nothing short of a whole share reads as a hundred.
 */
export const percentWord = (share: number): string => {
  if (share >= 1) return '100%'
  const whole = Math.floor(share * 100)
  return `${whole < 0 ? 0 : whole}%`
}

/**
 * How long the rest will take, on a clock.
 *
 * `perSecond` is measured by whoever is watching the count, because a rate needs
 * a clock and this has none. A rate of nothing means nothing is known, and
 * nothing is said: an estimate from no movement is a guess dressed as a fact.
 */
export const getRemainingWord = (left: number, perSecond: number): string => {
  if (left <= 0 || perSecond <= 0) return ''
  return clock((left / perSecond) * 1000)
}

/** How long a rate is averaged over, in seconds. */
export const SMOOTHING = 10

/**
 * The rate a count is moving at, from two readings and the time between them.
 *
 * `seconds` is the time since the count last moved, so a count written in
 * groups is measured over the stretch a group took.
 *
 * Movement is smoothed towards what was known over that stretch, so a reading
 * taken a moment after the last counts for a moment and one taken a minute later
 * counts for a minute.
 *
 * A count that went backwards is a fresh start, and has no rate until it is read
 * twice.
 */
export const rateOf = (
  previous: { readonly done: number; readonly rate: number },
  count: number,
  seconds: number,
): number => {
  if (seconds <= 0) return previous.rate
  const moved = count - previous.done
  if (moved < 0) return 0
  if (moved === 0) return previous.rate
  const now = moved / seconds
  if (previous.rate <= 0) return now
  return previous.rate + (1 - Math.exp(-seconds / SMOOTHING)) * (now - previous.rate)
}
