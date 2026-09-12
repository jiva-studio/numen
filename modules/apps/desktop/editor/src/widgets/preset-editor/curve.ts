/**
 * The one control of a preset, without drawing it: what the curve costs at
 * each place, where the knob stands, and the settings a place produces.
 *
 * The application works a curve out over the whole range in one pass, so moving
 * the control computes nothing. The line drawn while that answer is on its way
 * is arithmetic over the settings alone, and is shown as an approximation.
 */
import { dayAfter, daysBetween } from '@numen/ui'
import { DEFAULTS, NOWHERE } from './api/core'
import type {
  Bounds,
  Curve,
  Goal,
  Place,
  Point,
  Rule,
  Settings,
  SettingsBounds,
} from './api/core'

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

/** One setting of a preset a person may take out of the goal's hands. */
export type Field =
  | 'minutesADay'
  | 'newADay'
  | 'reviewsADay'
  | 'retention'
  | 'learned'
  | 'interval'
  | 'byDate'
  | 'counts'
  | 'backlog'
  | 'load'
  | 'evenLoad'

/** Every setting the receipt has a row for, in the order they stand in. */
export const FIELDS: readonly Field[] = [
  'newADay',
  'reviewsADay',
  // The rule for what is learned stands over the value it reads, and one of
  // those values is the target, so the rule stands over that too.
  'learned',
  'interval',
  'retention',
  'minutesADay',
  'byDate',
  'counts',
  'backlog',
  'load',
  'evenLoad',
]

/**
 * The budget each goal schedules by. A goal names one, and the settings of the
 * other two take no part in it: they are neither drawn nor written while it
 * stands, and the file keeps them where the person left them.
 *
 * Minutes are a budget of time. Retention is a budget of cards, since the
 * counts are what close a day worked to a target. A date is neither: the pace
 * follows from the day named, and nothing else may cut it short.
 */
const BUDGETS: Record<Goal, readonly Field[]> = {
  minutes: ['minutesADay'],
  // What a day's budget is spent on belongs to the goal whose budget is cards.
  retention: ['retention', 'newADay', 'reviewsADay', 'counts'],
  date: ['byDate'],
}

/**
 * The share of a day that goes to the debt before anything new is offered. It
 * says what a day is spent on and closes nothing, so it stands beside a budget
 * and not among them. A goal of a date carries the whole material by its own
 * reckoning and has no part in it.
 */
const SPENDING: Record<Goal, readonly Field[]> = {
  minutes: ['backlog'],
  retention: ['backlog'],
  date: [],
}

/**
 * What each rule for the learned reads. The rule itself is drawn under every
 * goal, and under it the one value that rule reads; the other keeps its value
 * and takes no part, as a budget the goal does not name does.
 *
 * A target is read by the goal that steers it and by the rule that counts by
 * it, and either way it is the one key: the receipt draws it once.
 */
const LEARNS: Record<Rule, readonly Field[]> = {
  interval: ['learned', 'interval'],
  retention: ['learned', 'retention'],
}

/** The settings that stand under no goal in particular, and are drawn under all. */
const ALWAYS: readonly Field[] = ['load', 'evenLoad']

/** The settings a goal schedules by, which are the rows the receipt draws. */
export const fieldsUnder = (goal: Goal, learned: Rule): readonly Field[] => {
  const drawn = new Set<Field>([
    ...BUDGETS[goal],
    ...SPENDING[goal],
    ...LEARNS[learned],
    ...ALWAYS,
  ])
  return FIELDS.filter((field) => drawn.has(field))
}

/** The field the goal steers, which is the knob under another name. */
export const steer = (goal: Goal): Field => {
  if (goal === 'minutes') return 'minutesADay'
  if (goal === 'retention') return 'retention'
  return 'byDate'
}

/**
 * The settings that give a curve its shape, as one word. The value the knob
 * rides is among them: the range a curve is drawn over runs to it. A day
 * steers nothing unless it is the goal, so it is left out of the other two:
 * choosing a date and coming back does not make their curves worth asking for
 * again.
 */
export const shapeOf = (settings: Settings): string => {
  const own = new Set<Field>(steer(settings.goal) === 'byDate' ? [] : ['byDate'])
  const said: Record<Field, string> = {
    newADay: `${settings.newADay}`,
    reviewsADay: `${settings.reviewsADay}`,
    retention: `${settings.retention}`,
    minutesADay: `${settings.minutesADay}`,
    byDate: settings.byDate,
    counts: settings.counts,
    backlog: `${settings.backlog}`,
    learned: settings.learned,
    interval: `${settings.interval}`,
    load: Object.keys(settings.load)
      .sort()
      .map((day) => `${day}=${settings.load[day]}`)
      .join(','),
    evenLoad: `${settings.evenLoad}`,
  }
  const rest = FIELDS.filter((field) => !own.has(field)).map((field) => `${field}=${said[field]}`)
  return [settings.goal, ...rest].join(' ')
}

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
export const nearest = (grid: readonly number[], value: number): number => {
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
  if (!curve.honest) return ''
  if (curve.decks === 0) return 'unpointed'
  if (curve.cards === 0) return 'noCards'
  if (curve.unbegun === curve.cards && asksNothing(curve)) return 'beginsNothing'
  return ''
}

/** Whether no place of the range asks for a card. A range with no place says nothing. */
const asksNothing = (curve: Curve): boolean =>
  curve.at.length > 0 && curve.at.every((one) => one.reviews === 0)

/**
 * The line the window draws while the application is still working the honest
 * one out. It is arithmetic over the settings alone, and it says so.
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
  const at = grid.map((value) => guessed(settings, value, grid))
  const days = settings.goal === 'date' ? grid.map((value) => dayAfter(today, value)) : []
  const value = goalValue(settings, today)
  const place = nearest(grid, value)
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
    honest: false,
  }
}

/** The goal's own value in the preset, in the units of its grid. */
export const goalValue = (settings: Settings, today: string): number => {
  if (settings.goal === 'retention') return settings.retention
  if (settings.goal === 'date') return daysBetween(today, settings.byDate)
  return settings.minutesADay
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
const guessed = (settings: Settings, value: number, grid: readonly number[]): Point => {
  const flat = {
    retained: 0,
    owed: 0,
    through: 0,
    enough: true,
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
    const enough = minutes <= (settings.minutesADay || DEFAULTS.minutesADay)
    const through = Math.min(value / span, 1)
    return { ...flat, minutes, reviews: (minutes * 60) / ANSWER, through, enough }
  }
  const retained = (CEILING * value) / (value + HALF)
  return { ...flat, minutes: value, reviews: (value * 60) / ANSWER, retained }
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
