import type { StepStatus } from './timeline.js'

export type StepStarts = readonly (number | null)[]

export type StepPhase = 'done' | 'current' | 'ahead'

export type Segment = {
  readonly index: number
  readonly left: number
  readonly width: number
}

export const findStepAt = (starts: StepStarts, time: number): number => {
  let found = -1
  starts.forEach((start, index) => {
    if (start !== null && start <= time) found = index
  })
  return found
}

export const findNextStart = (starts: StepStarts, time: number): number | null => {
  const after = time + 0.05
  for (const start of starts) {
    if (start !== null && start > after) return start
  }
  return null
}

export const findPreviousStart = (starts: StepStarts, time: number): number => {
  const started = starts.filter((start): start is number => start !== null && start <= time)
  const current = started.at(-1)
  if (current === undefined) return 0
  if (time - current > 1) return current
  return started.at(-2) ?? 0
}

export const getStepPhases = (count: number, currentIndex: number): StepPhase[] =>
  Array.from({ length: count }, (_, index) => {
    if (index < currentIndex) return 'done'
    return index === currentIndex ? 'current' : 'ahead'
  })

export const getSegments = (starts: StepStarts, length: number): Segment[] => {
  if (!(length > 0)) return []
  const ran = starts.flatMap((start, index) => (start === null ? [] : [{ index, start }]))
  return ran.flatMap(({ index, start }, position) => {
    const left = Math.min((start * 100) / length, 100)
    const end = Math.min(((ran[position + 1]?.start ?? length) * 100) / length, 100)
    return end > left ? [{ index, left, width: end - left }] : []
  })
}

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

export const getSetupLength = (steps: readonly { readonly keyword: string }[]): number => {
  const index = steps.findIndex((step) => step.keyword === 'When' || step.keyword === 'Then')
  return index === -1 ? steps.length : index
}

export const findActionStart = (starts: StepStarts, setupLength: number): number | null =>
  starts.slice(setupLength).find((start) => start !== null) ?? null

export const getWorstStatus = (statuses: readonly StepStatus[]): StepStatus => {
  if (statuses.includes('failed')) return 'failed'
  return statuses.includes('skipped') ? 'skipped' : 'passed'
}
