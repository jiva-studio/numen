/**
 * A year of days as a grid of weeks, laid out to the width there is for it.
 *
 * Apart from the drawing because how many days fit, which day each cell is and
 * how dark it is drawn are arithmetic, and arithmetic inside a component is
 * arithmetic nobody can check without a screen.
 */

/** How many days stand in one column. A column is a week. */
export const ROWS = 7

/** One day of the grid. */
export interface Day {
  /** The day it is, as the year, the month and the day it began on. */
  readonly day: string
  /** How much was done on it, or how much falls on it where it is still ahead. */
  readonly did: number
  /** How dark it is drawn: nothing at 0, most at 4. */
  readonly weight: 0 | 1 | 2 | 3 | 4
  /** Whether it is the day holding now. */
  readonly today: boolean
  /** Whether it is still to come, and what it holds is what is coming. */
  readonly ahead: boolean
}

/** How many weeks of what is still to come the grid keeps room for. */
export const AHEAD = 4

/** What a grid is laid out to. */
export interface Room {
  /** How wide the grid may be, in pixels. */
  width: number
  /** How large one cell is drawn, at most. */
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
export function fits(room: Room): { columns: number; cell: number; gap: number } {
  const cell = Math.max(1, room.cell)
  const gap = Math.max(0, room.gap)
  const step = cell + gap
  if (room.width <= 0) return { columns: 1, cell, gap }

  const columns = Math.max(1, Math.floor((room.width + gap) / step))
  if (columns < 2) return { columns, cell, gap }

  // The room the cells do not take is the room between them.
  const between = Math.max(gap, (room.width - columns * cell) / (columns - 1))
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
export function days(
  columns: number,
  now: Date,
  did: ReadonlyMap<string, number>,
  due: ReadonlyMap<string, number> = new Map(),
  named: (at: Date) => string = names,
): Day[] {
  const out: Day[] = []
  if (columns < 1) return out

  const today = named(now)
  // The last day drawn: the Sunday ending the last week kept for what is
  // still to come, or the week today stands in where there is no room for more.
  const weeks = Math.min(AHEAD, Math.max(0, columns - 1))
  const last = new Date(now)
  last.setHours(12, 0, 0, 0)
  last.setDate(last.getDate() + ((7 - weekday(last)) % 7) + weeks * ROWS)

  const first = new Date(last)
  first.setDate(first.getDate() - (columns * ROWS - 1))

  for (let at = 0; at < columns * ROWS; at += 1) {
    const on = new Date(first)
    on.setDate(on.getDate() + at)
    const day = named(on)
    const ahead = day > today
    const count = (ahead ? due.get(day) : did.get(day)) ?? 0
    out.push({ day, did: count, weight: weighs(count), today: day === today, ahead })
  }
  return out
}

/** Monday is the first day of a week, and Sunday the seventh. */
function weekday(at: Date): number {
  const day = at.getDay()
  return day === 0 ? 7 : day
}

/**
 * How dark a day is drawn. The steps are small on purpose: a person who
 * answered five cards did sit down, and the grid says so as plainly as it says
 * a day of fifty.
 */
export function weighs(did: number): Day['weight'] {
  if (did <= 0) return 0
  if (did < 5) return 1
  if (did < 20) return 2
  if (did < 50) return 3
  return 4
}

/** A day as it is written down: the year, the month and the day. */
export function names(at: Date): string {
  const month = String(at.getMonth() + 1).padStart(2, '0')
  const day = String(at.getDate()).padStart(2, '0')
  return `${at.getFullYear()}-${month}-${day}`
}
