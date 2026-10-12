/**
 * The room a grid of days is laid out in: how many columns fit it, and the
 * stretch of time a grid that wide draws.
 */
import { getDayName } from '@/shared/lib/day'

/** How many days stand in one column. A column is a week. */
export const ROWS = 7

/** How many weeks of what is still to come the grid keeps room for. */
export const AHEAD = 4

/** What a grid is laid out to. */
export interface HeatmapMetrics {
  /** How wide the grid may be, in pixels. */
  width: number
  /** How large one cell is drawn. */
  cell: number
  /** How much room is left between two cells. */
  gap: number
}

/**
 * How many columns fit the room there is, and how large a cell is drawn in it.
 *
 * The cell has a size of its own and the grid takes as many columns as fit, so
 * a wide window shows more weeks rather than the same weeks drawn larger, and a
 * narrow one shows fewer rather than the grid standing in the middle of empty
 * room. What is left over is spread between the cells, which keeps the grid
 * flush to both edges.
 */
export function measureGrid(metrics: HeatmapMetrics): {
  columns: number
  cell: number
  gap: number
} {
  const cell = Math.max(1, metrics.cell)
  const gap = Math.max(0, metrics.gap)
  const step = cell + gap
  if (metrics.width <= 0) return { columns: 1, cell, gap }

  const columns = Math.max(1, Math.floor((metrics.width + gap) / step))
  if (columns < 2) return { columns, cell, gap }

  // The room the cells do not take is the room between them.
  const between = Math.max(gap, (metrics.width - columns * cell) / (columns - 1))
  return { columns, cell, gap: between }
}

/**
 * The days a grid of this many columns draws, oldest first.
 *
 * The weeks behind a person run up to the one they are in, and a few weeks of
 * what is still to come stand after it, so the grid says what is coming as well
 * as what was done. Every column is a whole week.
 *
 * A day still to come holds what falls on it; a day behind holds what was
 * answered on it. Today holds what was answered, because that is the number a
 * person is adding to.
 */
/**
 * The stretch of time a grid this wide draws, oldest day first, as the names a
 * day is written under.
 *
 * A caller asks the application for the days before it knows the room it will
 * have, so it asks about the widest grid the room could hold. Every day any
 * narrower grid draws stands inside the answer.
 */
export function getStretch(metrics: HeatmapMetrics, now: Date): { from: string; to: string } {
  const { columns } = measureGrid(metrics)
  const weeks = Math.min(AHEAD, Math.max(0, columns - 1))

  // The grid running back from now: the Sunday ending the last week kept for
  // what is still to come, and the columns before it.
  const last = new Date(now)
  last.setHours(12, 0, 0, 0)
  last.setDate(last.getDate() + ((7 - weekday(last)) % 7) + weeks * ROWS)
  const behind = new Date(last)
  behind.setDate(behind.getDate() - (columns * ROWS - 1))

  // The grid opening on the week a person began in, which for a vault with
  // nothing behind it is this week, and then all of it stands ahead.
  const opens = monday(now)
  const ahead = new Date(opens)
  ahead.setDate(ahead.getDate() + columns * ROWS - 1)

  return {
    from: getDayName(behind < opens ? behind : opens),
    to: getDayName(last > ahead ? last : ahead),
  }
}

/** The Monday of the week a day stands in, which is the column it opens. */
function monday(at: Date): Date {
  const out = new Date(at)
  out.setHours(12, 0, 0, 0)
  out.setDate(out.getDate() - (weekday(out) - 1))
  return out
}

/** Monday is the first day of a week, and Sunday the seventh. */
function weekday(at: Date): number {
  const day = at.getDay()
  return day === 0 ? 7 : day
}
