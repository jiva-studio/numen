/** The place a row took in its list before a drag began. */
export interface RowSpan {
  readonly top: number
  readonly height: number
}

/** How far the held row may travel so that it stays inside the list. */
export function clampRowOffset(spans: readonly RowSpan[], from: number, delta: number): number {
  const held = spans[from]
  const first = spans[0]
  const last = spans[spans.length - 1]
  if (!held || !first || !last) return 0
  const min = first.top - held.top
  const max = last.top + last.height - held.height - held.top
  return Math.min(Math.max(delta, min), max)
}

const HYSTERESIS = 1 / 8

const getMiddle = (span: RowSpan): number => span.top + span.height / 2

/**
 * The place the held row is over, from where the rows stood when the drag began and how far the pointer
 * went. The row's centre must pass the middle of a row to take its place, and must go back past the middle
 * by a margin to give it up, so a pointer resting on the boundary does not flip between two places.
 */
export function getDropIndex(
  spans: readonly RowSpan[],
  from: number,
  delta: number,
  current: number = from,
): number | null {
  const held = spans[from]
  if (!held) return null
  if (delta >= clampRowOffset(spans, from, Infinity)) return spans.length - 1
  if (delta <= clampRowOffset(spans, from, -Infinity)) return 0
  const centre = getMiddle(held) + delta
  const margin = held.height * HYSTERESIS
  const isPassed = (at: number, isBelow: boolean): boolean => {
    const span = spans[at]
    if (!span) return false
    const isHeldBy = isBelow ? current >= at : current <= at
    const reach = isHeldBy ? -margin : margin
    return isBelow ? centre > getMiddle(span) + reach : centre < getMiddle(span) - reach
  }
  let target = from
  while (isPassed(target + 1, true)) target += 1
  if (target !== from) return target
  while (isPassed(target - 1, false)) target -= 1
  return target
}

/** How far each row of a list is moved aside while the row at `from` is held over the place `to`. */
export function getRowShifts(
  count: number,
  from: number,
  to: number,
  pitch: number,
): readonly number[] {
  return Array.from({ length: count }, (_, at) => {
    if (at === from) return 0
    if (from < to && at > from && at <= to) return -pitch
    if (from > to && at >= to && at < from) return pitch
    return 0
  })
}

/** How far the held row's place moves when it lands over `to`: the rows between slide, so its new top is not its old one. */
export function getLandingShift(spans: readonly RowSpan[], from: number, to: number): number {
  const held = spans[from]
  const target = spans[to]
  if (!held || !target || from === to) return 0
  const newTop = to > from ? target.top + target.height - held.height : target.top
  return newTop - held.top
}
