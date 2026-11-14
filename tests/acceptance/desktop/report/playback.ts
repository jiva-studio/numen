/**
 * How the page moves through the video of a scenario by its steps. The page runs these functions as
 * written here (write-report.ts writes their source into the page), so each one uses nothing from
 * outside it, not even another function of this file.
 */

import type { StepStatus } from './timeline.js'

/** Seconds from the start of the video at which each step started; null for a step that did not run. */
export type StepStarts = readonly (number | null)[]

export type StepPhase = 'done' | 'current' | 'ahead'

export type Segment = {
  readonly index: number
  /** Percent of the video. */
  readonly left: number
  readonly width: number
}

/** The index of the step the video shows at `time` seconds: the last that started by then, or -1. */
export const findStepAt = (starts: StepStarts, time: number): number => {
  let found = -1
  starts.forEach((start, index) => {
    if (start !== null && start <= time) found = index
  })
  return found
}

/** The start of the first step after `time`, where playback pauses at each step; null after the last. */
export const findNextStart = (starts: StepStarts, time: number): number | null => {
  // A video paused at a step stands a few milliseconds past its start or before it.
  const after = time + 0.05
  for (const start of starts) {
    if (start !== null && start > after) return start
  }
  return null
}

/**
 * Where the previous-step key goes from `time`: back to the start of the current step once more
 * than a second of it played, otherwise to the start of the step before; 0 before the first step.
 */
export const findPreviousStart = (starts: StepStarts, time: number): number => {
  const started = starts.filter((start): start is number => start !== null && start <= time)
  const current = started.at(-1)
  if (current === undefined) return 0
  if (time - current > 1) return current
  return started.at(-2) ?? 0
}

/** Each step against the step the video shows: played, playing or still to come. */
export const getStepPhases = (count: number, currentIndex: number): StepPhase[] =>
  Array.from({ length: count }, (_, index) => {
    if (index < currentIndex) return 'done'
    return index === currentIndex ? 'current' : 'ahead'
  })

/**
 * The part of the timeline each step that ran takes: from its start to the next step's, or to the
 * end. Steps that start together, as the steps that set up a scenario do, leave the last of them,
 * the one the video shows.
 */
export const getSegments = (starts: StepStarts, length: number): Segment[] => {
  if (!(length > 0)) return []
  const ran = starts.flatMap((start, index) => (start === null ? [] : [{ index, start }]))
  return ran.flatMap(({ index, start }, position) => {
    const left = Math.min((start * 100) / length, 100)
    const end = Math.min(((ran[position + 1]?.start ?? length) * 100) / length, 100)
    return end > left ? [{ index, left, width: end - left }] : []
  })
}

/**
 * The length of the video in seconds. A video Playwright records may carry no duration, which the
 * browser reports as Infinity; the end of the last step stands in for it then.
 */
export const getVideoLength = (
  mediaDuration: number,
  steps: readonly { readonly start: number | null; readonly duration: number }[],
): number => {
  if (Number.isFinite(mediaDuration) && mediaDuration > 0) return mediaDuration
  return steps.reduce(
    (end, step) => (step.start === null ? end : Math.max(end, step.start + step.duration / 1000)),
    0,
  )
}

/**
 * How many leading steps set the scenario up: the Background and the scenario's `Given` steps with
 * the `And` and `But` after them, up to the first `When` or `Then`.
 */
export const getSetupLength = (steps: readonly { readonly keyword: string }[]): number => {
  const index = steps.findIndex((step) => step.keyword === 'When' || step.keyword === 'Then')
  return index === -1 ? steps.length : index
}

/** The start of the first step after the setup that ran; null when none ran. */
export const findActionStart = (starts: StepStarts, setupLength: number): number | null =>
  starts.slice(setupLength).find((start) => start !== null) ?? null

/** The status of a group of steps: failed if one failed, skipped if one was skipped, else passed. */
export const getWorstStatus = (statuses: readonly StepStatus[]): StepStatus => {
  if (statuses.includes('failed')) return 'failed'
  return statuses.includes('skipped') ? 'skipped' : 'passed'
}
