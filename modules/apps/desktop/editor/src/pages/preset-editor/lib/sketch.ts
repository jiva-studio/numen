/**
 * The line the window draws while the application is still working the honest
 * one out. It is arithmetic over the settings alone, and it says so.
 */
import { dayAfter, daysBetween } from '@numen/ui'
import { DEFAULTS, NOWHERE } from '../types'
import type { Curve, Place, Point, Settings, SettingsBounds } from '../types'
import { goalValue, findNearest } from './curve'

/** How many places the line drawn in the answer's place is worked out at. */
const PLACES = 25

/** The shortest day a curve of minutes runs to. */
const LEAST_CEILING = 60

/** How long one answer takes where nothing has been answered yet, in seconds. */
const ANSWER = 8

/** Where the saturating line of minutes reaches half of what it ever reaches. */
const HALF = 15

/** The most of the material any day of review brings back. */
const CEILING = 0.98

/** The retention the standing budget is read as buying. */
const MIDDLE = 0.9

/**
 * The sketch of a curve, drawn from the settings alone.
 *
 * `today` is the review day the window was told, which is the day the core
 * counts from. It is not read off a clock here: a day of review begins hours
 * past midnight, and a calendar day would count a date one day nearer for as
 * long as the two disagree.
 *
 * `within` is how far each setting goes, as the application answered it. A
 * sketch of a target is drawn across the span it says, so the guess and the
 * answer stand over one range.
 */
export const approximate = (settings: Settings, today: string, within: SettingsBounds): Curve => {
  const grid = gridFor(settings, today, within)
  const at = grid.map((value) => estimatePoint(settings, value, grid))
  const days = settings.goal === 'date' ? grid.map((value) => dayAfter(today, value)) : []
  const value = goalValue(settings, today)
  const place = findNearest(grid, value)
  const now: Place = { at: place, value, day: days[place] ?? '' }
  return {
    goal: settings.goal,
    grid,
    days,
    at,
    now,
    suggested: NOWHERE,
    decks: 0,
    cards: 0,
    overdue: 0,
    unbegun: 0,
    isHonest: false,
  }
}

/**
 * The whole range of a goal, at the places the line is drawn at. A target is
 * held to the span the application answered with, and a span it has said
 * nothing about is no range at all: the sketch draws no places, and the answer
 * brings the honest ones.
 */
const gridFor = (settings: Settings, today: string, within: SettingsBounds): readonly number[] => {
  if (settings.goal === 'retention') {
    const span = within.retention
    if (span === undefined) return []
    return ladder(span.least, span.most, (one) => Math.round(one * 1000) / 1000)
  }
  if (settings.goal === 'date') {
    const most = Math.max(daysBetween(today, settings.byDate), 30)
    return ladder(1, most, Math.round)
  }
  return ladder(0, Math.max(settings.minutesADay, LEAST_CEILING), Math.round)
}

/** The places between two ends, both ends among them. */
const ladder = (least: number, most: number, rounds: (one: number) => number): readonly number[] =>
  Array.from({ length: PLACES }, (_, at) => rounds(least + ((most - least) * at) / (PLACES - 1)))

/**
 * What one place of the range comes to, guessed from the settings. A day of
 * review brings back more of the material the longer it runs, and asking for
 * more of it back costs more of the day.
 */
const estimatePoint = (settings: Settings, value: number, grid: readonly number[]): Point => {
  const flat = {
    retained: 0,
    owed: 0,
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
    const through = Math.min(value / span, 1)
    return { ...flat, minutes, reviews: (minutes * 60) / ANSWER, through, canLearnEveryCard }
  }
  const retained = (CEILING * value) / (value + HALF)
  return { ...flat, minutes: value, reviews: (value * 60) / ANSWER, retained }
}
