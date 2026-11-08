/**
 * The place in a recording as a selector: `t` is [start, end] in seconds (W3C Media Fragments).
 */
import type { Selector } from '@numen/plugin-sdk'

/** The selector of one moment of a recording, given in milliseconds. */
export const createTimeSelector = (ms: number): Selector => {
  const seconds = Math.max(0, ms) / 1000
  return { t: [seconds, seconds] }
}

/** The [start, end] span in seconds a selector names, and undefined where it names none. */
export const getSelectorSpan = (
  selector: Selector | null | undefined,
): readonly [number, number] | undefined => {
  const t = selector?.t
  return Array.isArray(t) && typeof t[0] === 'number' && typeof t[1] === 'number'
    ? [t[0], t[1]]
    : undefined
}

/** The start a selector names, in milliseconds, and null where it names none. */
export const getSelectorStart = (selector: Selector | null | undefined): number | null => {
  const start = getSelectorSpan(selector)?.[0]
  return typeof start === 'number' && Number.isFinite(start) && start >= 0
    ? Math.round(start * 1000)
    : null
}
