/**
 * What the curve of a preset evaluates to, and where the knob relies on it: the
 * cost at a place, the value a place reads out, and the settings it produces.
 *
 * The application works a curve out over the whole range in one pass, so moving
 * the control computes nothing.
 */
import { dayAfter, daysBetween } from '../../lib/day'
import type { Bounds, Curve, Goal, Point, Settings, SettingsBounds } from '../types'

/**
 * What the curve of a goal is read in. A goal of minutes is read in the cards
 * a day answers, which is what a longer day buys; the other two are read in
 * the minutes they cost.
 */
export const getPointCost = (goal: Goal, point: Point): number =>
  goal === 'minutes' ? point.reviews : point.minutes

/**
 * The day the overdue backlog is cleared, computed from the same projection the
 * backlog is drawn from. Null means nothing is overdue, and -1 means the backlog
 * remains on the last projected day.
 */
export const clearBacklog = (backlog: readonly number[]): number | null => {
  if (!backlog.some((one) => one > 0)) return null
  const index = backlog.indexOf(0)
  return index < 0 ? -1 : index + 1
}

/**
 * Limits a value to the bounds of its setting. A setting without bounds returns the
 * value unchanged.
 */
export const clamp = (value: number, within: Bounds | undefined): number =>
  within === undefined ? value : Math.min(Math.max(value, within.min), within.max)

/** The place of the grid nearest a value, and the last one for an empty grid. */
export const findNearest = (grid: readonly number[], value: number): number => {
  if (grid.length === 0) return -1
  let nearestIndex = 0
  for (let i = 1; i < grid.length; i += 1) {
    if (Math.abs((grid[i] ?? 0) - value) < Math.abs((grid[nearestIndex] ?? 0) - value))
      nearestIndex = i
  }
  return nearestIndex
}

/**
 * Returns the value shown for the knob at a place of the grid. The place the knob
 * opens at is the grid place nearest the preset's own value, so while the knob
 * stays at that place the preset's own value is returned. At any other place the
 * grid value is returned and is exact.
 */
export const getKnobValue = (curve: Curve, place: number): number =>
  curve.now.index >= 0 && place === curve.now.index ? curve.now.value : (curve.grid[place] ?? 0)

/** Why a preset's goal has nothing to work on, and empty where it has. */
export type GoalIdleReason = 'unpointed' | 'noCards' | 'beginsNothing' | ''

/**
 * Whether the goal has nothing to work on, and why. The counts the curve carries
 * identify the first two: no deck points at the preset, or the decks that do hold
 * no cards in total. A preset holding cards never reports that it holds none.
 *
 * The third is a preset that schedules but has nothing it can schedule: every
 * review card is unbegun, and no place of the range begins one. It describes the
 * material and is not a reason the preset is stopped.
 */
export const getIdleReason = (curve: Curve): GoalIdleReason => {
  if (!curve.isHonest) return ''
  if (curve.decks === 0) return 'unpointed'
  if (curve.cards === 0) return 'noCards'
  if (curve.unbegun === curve.cards && hasNoReviews(curve)) return 'beginsNothing'
  return ''
}

/** Whether no place of the range asks for a card. An empty range returns false. */
const hasNoReviews = (curve: Curve): boolean =>
  curve.at.length > 0 && curve.at.every((one) => one.reviews === 0)

/** The goal's own value in the preset, in the units of its grid. */
export const getGoalValue = (settings: Settings, today: string): number => {
  if (settings.goal === 'retention') return settings.retention
  if (settings.goal === 'date') return daysBetween(today, settings.byDate)
  return settings.minutesADay
}

export interface ProduceScheduleOptions {
  curve: Curve
  today: string
  within: SettingsBounds
}

/**
 * The settings one place of the curve produces: the goal's own value taken from
 * the grid. Every other setting keeps its current value.
 */
export const produceSchedule = (
  was: Settings,
  place: number,
  options: ProduceScheduleOptions,
): Settings => {
  const { curve, today, within } = options
  const value = curve.grid[place] ?? getGoalValue(was, today)
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
