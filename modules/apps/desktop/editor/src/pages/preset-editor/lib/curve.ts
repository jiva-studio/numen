/**
 * What the curve of a preset comes to, and where the knob stands on it: the
 * cost at a place, the value a place reads out, and the settings it produces.
 *
 * The application works a curve out over the whole range in one pass, so moving
 * the control computes nothing.
 */
import { dayAfter, daysBetween } from '@numen/ui'
import type { Bounds, Curve, Goal, Point, Settings, SettingsBounds } from '../types'

/**
 * What the curve of a goal is read in. A goal of minutes is read in the cards
 * a day answers, which is what a longer day buys; the other two are read in
 * the minutes they cost.
 */
export const costOf = (goal: Goal, point: Point): number =>
  goal === 'minutes' ? point.reviews : point.minutes

/**
 * The day the overdue pile is gone, read off the very projection the backlog is
 * drawn from. Null is a place with nothing overdue to be gone at all, and -1
 * is a pile still standing on the last day projected.
 */
export const clearBacklog = (backlog: readonly number[]): number | null => {
  if (!backlog.some((one) => one > 0)) return null
  const at = backlog.indexOf(0)
  return at < 0 ? -1 : at + 1
}

/**
 * A number held inside the bounds of the setting it is. A setting the
 * application has said no bound for is held to none.
 */
export const clamp = (value: number, within: Bounds | undefined): number =>
  within === undefined ? value : Math.min(Math.max(value, within.least), within.most)

/** The place of the grid nearest a value, and the last one for an empty grid. */
export const findNearest = (grid: readonly number[], value: number): number => {
  if (grid.length === 0) return -1
  let at = 0
  for (let i = 1; i < grid.length; i += 1) {
    if (Math.abs((grid[i] ?? 0) - value) < Math.abs((grid[at] ?? 0) - value)) at = i
  }
  return at
}

/**
 * The value the knob stands at. A preset's own value need not sit on the grid,
 * and the place it opens at is the one nearest it, so while the knob has not
 * been moved off that place the preset's own value is what is said. A knob
 * walked anywhere else stands on a place, and the place is exact.
 */
export const valueAt = (curve: Curve, place: number): number =>
  curve.now.at >= 0 && place === curve.now.at ? curve.now.value : (curve.grid[place] ?? 0)

/** Why a preset's goal has nothing to work on, and empty where it has. */
export type IdleReason = 'unpointed' | 'noCards' | 'beginsNothing' | ''

/**
 * Whether the goal has nothing to work on, and why. The counts the curve
 * carries say the first two: no deck points here, or the decks that do hold
 * nothing between them. A preset holding cards is never told it holds none.
 *
 * The third is a preset that schedules and has nothing it can schedule: every
 * card face here is one nobody has begun, and no place of the range begins one.
 * It is a fact about the material, and not a reason the preset is stopped.
 */
export const idle = (curve: Curve): IdleReason => {
  if (!curve.isHonest) return ''
  if (curve.decks === 0) return 'unpointed'
  if (curve.cards === 0) return 'noCards'
  if (curve.unbegun === curve.cards && hasNoReviews(curve)) return 'beginsNothing'
  return ''
}

/** Whether no place of the range asks for a card. A range with no place says nothing. */
const hasNoReviews = (curve: Curve): boolean =>
  curve.at.length > 0 && curve.at.every((one) => one.reviews === 0)

/** The goal's own value in the preset, in the units of its grid. */
export const goalValue = (settings: Settings, today: string): number => {
  if (settings.goal === 'retention') return settings.retention
  if (settings.goal === 'date') return daysBetween(today, settings.byDate)
  return settings.minutesADay
}

/**
 * The settings one place of the curve produces, which is the goal's own value
 * off the grid. Every other setting stands as the person left it.
 */
export const produceSchedule = (
  was: Settings,
  place: number,
  curve: Curve,
  today: string,
  within: SettingsBounds,
): Settings => {
  const value = curve.grid[place] ?? goalValue(was, today)
  if (curve.goal === 'retention') {
    return { ...was, retention: clamp(round(value, 2), within.retention) }
  }
  if (curve.goal === 'date') {
    return { ...was, byDate: curve.days[place] ?? dayAfter(today, value) }
  }
  return { ...was, minutesADay: clamp(Math.round(value), within.minutesADay) }
}

/** A number to that many places. */
export const round = (value: number, places: number): number => {
  const scale = 10 ** places
  return Math.round(value * scale) / scale
}
