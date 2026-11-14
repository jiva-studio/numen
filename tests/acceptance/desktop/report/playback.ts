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

export type Click = { readonly t: number; readonly x: number; readonly y: number }

export type Ripple = { readonly x: number; readonly y: number; readonly progress: number }

export type Cursor = { readonly x: number; readonly y: number; readonly ripple: Ripple | null }

export const getCursorAt = (clicks: readonly Click[], time: number): Cursor | null => {
  const travelLength = 0.6
  const rippleLength = 0.4
  const ease = (share: number): number =>
    share < 0.5 ? 2 * share * share : 1 - (-2 * share + 2) ** 2 / 2
  const getShare = (from: Click, to: Click): number => {
    const start = Math.max(from.t, to.t - travelLength)
    if (to.t <= start) return 1
    return ease(Math.min(Math.max((time - start) / (to.t - start), 0), 1))
  }
  const getRipple = (click: Click | undefined): Ripple | null => {
    if (!click || time - click.t >= rippleLength) return null
    return { x: click.x, y: click.y, progress: (time - click.t) / rippleLength }
  }
  const nextIndex = clicks.findIndex((click) => click.t > time)
  const lastIndex = (nextIndex === -1 ? clicks.length : nextIndex) - 1
  const previous = lastIndex < 0 ? undefined : clicks.at(lastIndex)
  const next = nextIndex === -1 ? undefined : clicks.at(nextIndex)
  const from = previous ?? next
  if (!from) return null
  const share = previous && next ? getShare(previous, next) : 0
  const to = next ?? from
  return {
    x: from.x + (to.x - from.x) * share,
    y: from.y + (to.y - from.y) * share,
    ripple: getRipple(previous),
  }
}
