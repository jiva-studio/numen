/**
 * The provisional line drawn while the application computes the actual curve. It
 * is arithmetic over the settings alone.
 */
import { dayAfter, daysBetween } from '../../lib/day'
import {
  DEFAULTS,
  NO_CURVE_POINT,
  type Curve,
  type CurvePoint,
  type Point,
  type Settings,
  type SettingsBounds,
} from '../../lib/presets'
import { getGoalValue, findNearestGridValue } from './curve'

/** How many places the line drawn in the answer's place is worked out at. */
const PLACES = 25

/** The shortest day a curve of minutes runs to. */
const LEAST_CEILING = 60

/** How long one answer takes where nothing has been answered yet, in seconds. */
const ANSWER = 8

/** The value at which the saturating line of minutes attains half of its maximum. */
const HALF = 15

/** The most of the material any day of review brings back. */
const CEILING = 0.98

/** The retention the standing budget is read as buying. */
const MIDDLE = 0.9

/**
 * The sketch of a curve, drawn from the settings alone.
 * `today` is the review day supplied by the application, not read from a clock: a review day
 * begins hours after midnight, so a calendar day would count a date one day nearer.
 * `within` is the range of each setting; a target is sketched across that span so the estimate
 * and the actual curve share one range.
 */
export const estimateCurve = (settings: Settings, today: string, within: SettingsBounds): Curve => {
  const grid = createGrid(settings, today, within)
  const points = grid.map((value) => estimatePoint(settings, value, grid))
  const days = settings.goal === 'date' ? grid.map((value) => dayAfter(today, value)) : []
  const value = getGoalValue(settings, today)
  const place = findNearestGridValue(grid, value)
  const now: CurvePoint = { index: place, value, day: days[place] ?? '' }
  return {
    goal: settings.goal,
    grid,
    days,
    at: points,
    now,
    suggested: NO_CURVE_POINT,
    decks: 0,
    cards: 0,
    overdue: 0,
    unbegun: 0,
    isHonest: false,
  }
}

const createGrid = (
  settings: Settings,
  today: string,
  within: SettingsBounds,
): readonly number[] => {
  if (settings.goal === 'retention') {
    const span = within.retention
    if (span === undefined) return []
    return ladder(span.min, span.max, (one) => Math.round(one * 1000) / 1000)
  }
  if (settings.goal === 'date') {
    const most = Math.max(daysBetween(today, settings.byDate), 30)
    return ladder(1, most, Math.round)
  }
  return ladder(0, Math.max(settings.minutesADay, LEAST_CEILING), Math.round)
}

const ladder = (least: number, most: number, rounds: (one: number) => number): readonly number[] =>
  Array.from({ length: PLACES }, (_, index) =>
    rounds(least + ((most - least) * index) / (PLACES - 1)),
  )

const estimatePoint = (settings: Settings, value: number, grid: readonly number[]): Point => {
  const flat = {
    retained: 0,
    due: 0,
    through: 0,
    canLearnEveryCard: true,
    closed: [],
    clears: 0,
    learned: 0,
    short: 0,
    backlog: [],
  }
  if (settings.goal === 'retention') {
    const minutes = (settings.minutesADay || DEFAULTS.minutesADay) * ((1 - MIDDLE) / (1 - value))
    return { ...flat, minutes, reviews: (minutes * 60) / ANSWER, retained: value }
  }
  if (settings.goal === 'date') {
    const span = Math.max(grid[grid.length - 1] ?? 1, 1)
    const minutes = ((settings.minutesADay || DEFAULTS.minutesADay) * span) / Math.max(value, 1)
    const canLearnEveryCard = minutes <= (settings.minutesADay || DEFAULTS.minutesADay)
    return {
      ...flat,
      minutes,
      reviews: (minutes * 60) / ANSWER,
      through: Math.min(value / span, 1),
      canLearnEveryCard,
    }
  }
  const retentionRatio = (CEILING * value) / (value + HALF)
  return { ...flat, minutes: value, reviews: (value * 60) / ANSWER, retained: retentionRatio }
}
