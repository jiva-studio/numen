import { countColumns } from './columnCount'
import { getColumnPitch } from './columnPitch'
import type { Flow } from './spread'

/**
 * The narrowest a column is set, in CSS pixels at the size the text is read at.
 * A line of about fifty-six characters, which is within the measure prose is
 * set at;
 * a second column is opened only where both can be read at it.
 */
export const NARROWEST = 448

/** How wide one column is set: its share of the area, less the gap it keeps. */
export function getColumnWidth(flow: Flow): number {
  return Math.max(getColumnPitch(flow) - flow.gap, 0)
}

/**
 * The column that a horizontal position falls in, counted from the first. A
 * column begins half a gap into its share of the area. The browser lays out
 * columns in whole device pixels, so a run measured a fraction of a pixel short
 * of its own column is compared against that half gap.
 */
export function getColumn(flow: Flow, x: number): number {
  const pitch = getColumnPitch(flow)
  if (pitch <= 0) return 0
  return Math.max(Math.floor(x / pitch), 0)
}

/** How many countSpreads the text evaluates to. */
export function countSpreads(flow: Flow): number {
  const all = countColumns(flow)
  return all === 0 ? 0 : Math.ceil(all / flow.columns)
}

/** Which spread a place along the columns falls in. */
export function getSpread(flow: Flow, x: number): number {
  if (getColumnWidth(flow) <= 0) return 0
  const last = Math.max(countSpreads(flow) - 1, 0)
  return Math.min(Math.floor(getColumn(flow, x) / flow.columns), last)
}
