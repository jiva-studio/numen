/**
 * A year of days as a grid of weeks, laid out to the available width.
 *
 * Separate from the rendering because the number of days that fit, the date of
 * each cell and its shade are arithmetic, and arithmetic inside a component
 * cannot be tested without a screen.
 */
import { parseDay, getDayName } from '../../../lib/day'

import { AHEAD, ROWS } from './grid'

/** What one day behind came to. */
export interface Tally {
  readonly reviewed: number
  /** The number of answers per rating. */
  readonly again: number
  readonly hard: number
  readonly good: number
  readonly easy: number
  /**
   * The answers given to cards already under review, and how many of those were
   * recalled. A card still repeating within minutes is in neither count.
   */
  readonly tested: number
  readonly recalled: number
}

/** A day with no answers, which is the value of an empty cell. */
export const NOTHING: Tally = {
  reviewed: 0,
  again: 0,
  hard: 0,
  good: 0,
  easy: 0,
  tested: 0,
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
  readonly isToday: boolean
  /** Whether it is still to come, and what it holds is what is coming. */
  readonly isAhead: boolean
}

export interface HeatmapHistory {
  readonly did: ReadonlyMap<string, Tally>
  readonly due?: ReadonlyMap<string, number>
}

export function getDays(
  columns: number,
  now: Date,
  history: ReadonlyMap<string, Tally> | HeatmapHistory,
): Day[] {
  const [did, due] =
    'did' in history
      ? [history.did, history.due ?? new Map<string, number>()]
      : [history, new Map<string, number>()]
  const out: Day[] = []

  const today = getDayName(now)

  // The last day the grid would draw if it ran back from now: the Sunday ending
  // the last week kept for what is still to come.
  const weeks = Math.min(AHEAD, Math.max(0, columns - 1))
  const last = new Date(now)
  last.setDate(last.getDate() + ((7 - weekday(last)) % 7) + weeks * ROWS)

  // The grid takes the width it is given and starts on the week of the first
  // recorded day: its days run from the left, and the space after the last day
  // extends to the right. Once the history is longer than the width, the oldest
  // weeks are dropped from the left.
  const startDate = new Date(last)
  startDate.setDate(startDate.getDate() - (columns * ROWS - 1))

  const opens = monday(getFirstDay(did, due, now))
  const first = opens > startDate ? opens : startDate

  for (let offset = 0; offset < columns * ROWS; offset += 1) {
    const on = new Date(first)
    on.setDate(on.getDate() + offset)
    const day = getDayName(on)
    const isAhead = day > today
    const tally = did.get(day) ?? NOTHING
    const count = isAhead ? (due.get(day) ?? 0) : tally.reviewed
    out.push({
      ...(isAhead ? NOTHING : tally),
      day,
      did: count,
      weight: getWeight(count),
      isToday: day === today,
      isAhead,
    })
  }
  return out
}

/**
 * The first day of the history, or today when there is none. A day still to come
 * counts: a vault whose cards are all in the future has a first day too.
 */
function getFirstDay(
  did: ReadonlyMap<string, unknown>,
  due: ReadonlyMap<string, unknown>,
  now: Date,
): Date {
  let first = ''
  for (const day of [...did.keys(), ...due.keys()]) {
    if (first === '' || day < first) first = day
  }
  if (first === '') return now
  const date = parseDay(first)
  return date > now ? now : date
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

/**
 * The shade of a day. The steps are small: five answered cards is a day of
 * activity, and the grid shows it as clearly as a day of fifty.
 */
export function getWeight(did: number): Day['weight'] {
  if (did <= 0) return 0
  if (did < 5) return 1
  if (did < 20) return 2
  if (did < 50) return 3
  return 4
}

/** How round the corner of a cell is drawn, in the grid's own units. */
const RADIUS = 2

export interface Cell {
  readonly day: Day
  readonly x: number
  readonly y: number
  readonly size: number
  readonly radius: number
}

/** How far apart two cells begin. */
const getStep = (grid: { cell: number; gap: number }): number => grid.cell + grid.gap

/** The grid height: seven rows, with no gap after the last one. */
export function getGridHeight(grid: { cell: number; gap: number }): number {
  return ROWS * getStep(grid) - grid.gap
}

/**
 * Where every day is drawn. A column is a week, so a day's place is its own
 * number over the rows and the remainder down them.
 */
export function getCells(
  grid: { cell: number; gap: number },
  days: readonly Day[],
): readonly Cell[] {
  const step = getStep(grid)
  return days.map((day, index) => ({
    day,
    x: Math.floor(index / ROWS) * step,
    y: (index % ROWS) * step,
    size: grid.cell,
    radius: RADIUS,
  }))
}
