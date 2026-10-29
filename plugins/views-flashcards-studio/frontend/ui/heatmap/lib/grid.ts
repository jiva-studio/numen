/**
 * Layout of a grid of days: the number of columns that fit the available width,
 * and the date range a grid of that width displays.
 */
import { getDayName } from '../../../lib/day'

/** How many days stand in one column. A column is a week. */
export const ROWS = 7

/** The number of future weeks the grid reserves columns for. */
export const AHEAD = 4

/** What a grid is laid out to. */
export interface HeatmapMetrics {
  /** How wide the grid may be, in pixels. */
  width: number
  /** How large one cell is drawn. */
  cell: number
  /** Space between two cells. */
  gap: number
}

/**
 * The number of columns that fit the available width, and the cell size used.
 *
 * The cell size is fixed and the number of columns follows from the width, so a
 * wide window shows more weeks with the same cell size, and a narrow window
 * shows fewer weeks without empty space around the grid. The remaining width
 * is distributed between the cells, which aligns the grid to both edges.
 */
export function measureGrid(metrics: HeatmapMetrics): {
  columns: number
  cell: number
  gap: number
} {
  const cell = Math.max(1, metrics.cell)
  const gap = Math.max(0, metrics.gap)
  const step = cell + gap

  const columns = Math.max(1, Math.floor((metrics.width + gap) / step))
  if (columns < 2) return { columns, cell, gap }

  // The width not used by cells is distributed between them.
  const between = Math.max(gap, (metrics.width - columns * cell) / (columns - 1))
  return { columns, cell, gap: between }
}

/**
 * The days a grid of this many columns displays, oldest first.
 *
 * The weeks before today run up to the current week, and a few future weeks
 * follow it, so the grid shows the schedule ahead as well as the history. Every
 * column is a whole week.
 *
 * A future day holds the number of cards due on it; a past day holds the number
 * answered on it. Today holds the number answered, because that is the count
 * that increases during the day.
 */
/**
 * The date range a grid of this width displays, oldest day first, as day names.
 *
 * A caller requests the days before the available width is known, so it requests
 * the range of the widest grid the width could hold. Every day that any narrower
 * grid displays is inside that range.
 */
export function getStretch(metrics: HeatmapMetrics, now: Date): { from: string; to: string } {
  const { columns } = measureGrid(metrics)
  const weeks = Math.min(AHEAD, Math.max(0, columns - 1))

  // The grid running back from now: the Sunday ending the last week kept for
  // what is still to come, and the columns before it.
  const last = new Date(now)
  last.setDate(last.getDate() + ((7 - weekday(last)) % 7) + weeks * ROWS)
  const startDate = new Date(last)
  startDate.setDate(startDate.getDate() - (columns * ROWS - 1))

  // The grid opened on the current week, which is the case for a vault with no
  // history: every column is in the future and this is the furthest end the range
  // can have.
  const opens = monday(now)
  const ahead = new Date(opens)
  ahead.setDate(ahead.getDate() + columns * ROWS - 1)

  return { from: getDayName(startDate), to: getDayName(ahead) }
}

function monday(date: Date): Date {
  const out = new Date(date)
  out.setDate(out.getDate() - (weekday(out) - 1))
  return out
}

/** Monday is the first day of a week, and Sunday the seventh. */
function weekday(date: Date): number {
  const day = date.getDay()
  return day === 0 ? 7 : day
}
