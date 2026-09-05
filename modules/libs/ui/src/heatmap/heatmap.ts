/**
 * A year of days as a grid of weeks, laid out to the width there is for it.
 *
 * Apart from the drawing because how many days fit, which day each cell is and
 * how dark it is drawn are arithmetic, and arithmetic inside a component is
 * arithmetic nobody can check without a screen.
 */
import { dayNamed, dayOf } from '../calendar/day'

/** How many days stand in one column. A column is a week. */
export const ROWS = 7

/** What one day behind came to. */
export interface Tally {
  readonly answered: number
  /** How each of the four was said. */
  readonly again: number
  readonly hard: number
  readonly good: number
  readonly easy: number
  /**
   * The answers given to cards the person is already reviewing, and how many of
   * those came back. A card still coming round in minutes is in neither.
   */
  readonly asked: number
  readonly recalled: number
}

/** A day nobody answered on, which is what an empty cell holds. */
export const NOTHING: Tally = {
  answered: 0,
  again: 0,
  hard: 0,
  good: 0,
  easy: 0,
  asked: 0,
  recalled: 0,
}

/** One day of the grid. */
export interface Day extends Tally {
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
  did: ReadonlyMap<string, Tally>,
  due: ReadonlyMap<string, number> = new Map(),
  named: (at: Date) => string = dayNamed,
): Day[] {
  const out: Day[] = []
  if (columns < 1) return out

  const today = named(now)

  // The last day the grid would draw if it ran back from now: the Sunday ending
  // the last week kept for what is still to come.
  const weeks = Math.min(AHEAD, Math.max(0, columns - 1))
  const last = new Date(now)
  last.setHours(12, 0, 0, 0)
  last.setDate(last.getDate() + ((7 - weekday(last)) % 7) + weeks * ROWS)

  // The grid takes the width it is given, and it opens on the week a person
  // began in: their days are read from the left, and the room past what they
  // have yet done stretches out to the right. Once they have been here longer
  // than the width holds, the oldest weeks fall off the left instead.
  const behind = new Date(last)
  behind.setDate(behind.getDate() - (columns * ROWS - 1))

  const opens = monday(began(did, due, now))
  const first = opens > behind ? opens : behind

  for (let at = 0; at < columns * ROWS; at += 1) {
    const on = new Date(first)
    on.setDate(on.getDate() + at)
    const day = named(on)
    const ahead = day > today
    const tally = did.get(day) ?? NOTHING
    const count = ahead ? (due.get(day) ?? 0) : tally.answered
    out.push({
      ...(ahead ? NOTHING : tally),
      day,
      did: count,
      weight: weighs(count),
      today: day === today,
      ahead,
    })
  }
  return out
}

/**
 * The day a person's history begins, or today where they have none. A day still
 * to come counts: a vault whose cards are all ahead has a beginning too.
 */
function began(
  did: ReadonlyMap<string, unknown>,
  due: ReadonlyMap<string, unknown>,
  now: Date,
): Date {
  let first = ''
  for (const day of [...did.keys(), ...due.keys()]) {
    if (first === '' || day < first) first = day
  }
  if (first === '') return now
  const at = dayOf(first)
  return at > now ? now : at
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

