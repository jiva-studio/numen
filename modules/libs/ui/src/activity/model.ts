/**
 * What a line of activity is, as plain values. No DOM, no clock, no
 * measurement.
 *
 * Activity is work that runs whether or not anyone asked: a count that is
 * climbing, and a name for what is climbing it. Whoever renders this decides
 * what the words are, so nothing here knows what is being counted.
 */

import { grouped } from '../counting'

/**
 * Where a piece of work has got to.
 *
 * `resting` has words and no work: something worth knowing that nothing is
 * going to change by itself. It is drawn without the marks of progress.
 */
export type ActivityState = 'quiet' | 'working' | 'resting' | 'trouble'

/** A count of things done out of things to do. */
export interface Tally {
  readonly done: number
  readonly total: number
}

/**
 * What a count counts.
 *
 * Bytes are read out in the sizes a person reads them in. Everything else is
 * counted one by one.
 */
export type Counting = 'things' | 'bytes'

/** What a line of activity draws. */
export interface ActivityDescriptor {
  readonly state: ActivityState
  /** Whether a bar is drawn, and how full. Absent when the share is unknown. */
  readonly share?: number
  /** Whether the count is worth showing beside the words. */
  readonly counts: boolean
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
 * Trouble outranks everything: a line that is both failing and counting says it
 * is failing. Nothing to say is quiet, and quiet draws nothing — which is not
 * the same as finished. Everything else is resting or working, and a count is
 * what makes the difference visible.
 */
export const activity = (input: {
  readonly says: string
  readonly trouble?: boolean
  readonly working?: boolean
  readonly tally?: Tally
}): ActivityDescriptor => {
  if (input.trouble) return { state: 'trouble', counts: false }
  if (!input.says) return { state: 'quiet', counts: false }

  // Work is claimed, not assumed. Words alone say something is so, and a caller
  // that means "this is happening now" says that too.
  const share = input.tally ? shareOf(input.tally) : undefined
  if (!input.working && share === undefined) return { state: 'resting', counts: false }

  return share === undefined
    ? { state: 'working', counts: false }
    : { state: 'working', share, counts: true }
}

/**
 * A size in the units it is read in, in the thousands a machine reports its own
 * disk in.
 *
 * Whole units above a kilobyte: a figure with decimals in it changes every time
 * it is drawn, and a number that never settles reads as noise.
 */
export const sizeWord = (bytes: number): string => {
  const size = bytes < 0 ? 0 : bytes
  if (size < 1e3) return `${Math.round(size)} B`
  if (size < 1e6) return `${Math.round(size / 1e3)} kB`
  if (size < 1e9) return `${Math.round(size / 1e6)} MB`
  return `${(size / 1e9).toFixed(1)} GB`
}

/**
 * A tally as it is read out.
 *
 * Grouped in thousands, because the numbers this draws are counts of text and
 * reach six figures on an ordinary vault. A count that overtook its total reads
 * as the total: a vault loses a book mid-scan, and the bar is already full.
 */
export const tallyWord = (tally: Tally, counting: Counting = 'things'): string => {
  const done = Math.min(tally.done, tally.total)
  return counting === 'bytes'
    ? `${sizeWord(done)} of ${sizeWord(tally.total)}`
    : `${grouped(done)} of ${grouped(tally.total)}`
}

/**
 * How fast a count is moving, in words.
 *
 * `perSecond` is measured by whoever is watching the count. A rate of nothing
 * is nothing known, and nothing is said.
 */
export const rateWord = (perSecond: number, counting: Counting = 'things'): string => {
  if (perSecond <= 0) return ''
  return counting === 'bytes' ? `${sizeWord(perSecond)}/s` : `${grouped(Math.round(perSecond))}/s`
}


/**
 * A share as a percentage, for reading beside the count.
 *
 * Rounded down, so that nothing says a hundred per cent until it is finished.
 * A share of exactly one is the only way to read a hundred.
 */
export const percentWord = (share: number): string => {
  if (share >= 1) return '100%'
  const whole = Math.floor(share * 100)
  return `${whole < 0 ? 0 : whole}%`
}

/**
 * How long the rest will take, in words.
 *
 * `perSecond` is measured by whoever is watching the count, because a rate needs
 * a clock and this has none. A rate of nothing means nothing is known, and
 * nothing is said: an estimate from no movement is a guess dressed as a fact.
 *
 * Coarse on purpose. Work measured in hours does not become more predictable by
 * being reported to the minute, and a figure that jitters every time it is drawn
 * reads as broken.
 */
export const remainingWord = (left: number, perSecond: number): string => {
  if (left <= 0 || perSecond <= 0) return ''
  const seconds = left / perSecond
  if (seconds < 60) return 'under a minute left'
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `about ${minutes} ${minutes === 1 ? 'minute' : 'minutes'} left`
  const hours = Math.round(seconds / 3600)
  if (hours < 24) return `about ${hours} ${hours === 1 ? 'hour' : 'hours'} left`
  const days = Math.round(seconds / 86400)
  return `about ${days} ${days === 1 ? 'day' : 'days'} left`
}

/**
 * The rate a count is moving at, from two readings and the time between them.
 *
 * A count written in groups stands still between them, and a reading that saw no
 * movement measured nothing: the rate already known stands, and what is drawn
 * from it stands with it. Movement is smoothed towards what was known, so one
 * group arriving at once does not become the rate.
 *
 * A count that went backwards is a fresh start, and has no rate until it is read
 * twice.
 */
export const rateOf = (
  previous: { readonly done: number; readonly rate: number },
  done: number,
  seconds: number,
): number => {
  if (seconds <= 0) return previous.rate
  const moved = done - previous.done
  if (moved < 0) return 0
  if (moved === 0) return previous.rate
  const now = moved / seconds
  if (previous.rate <= 0) return now
  return previous.rate * 0.7 + now * 0.3
}
