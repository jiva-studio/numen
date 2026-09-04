/**
 * The one control of a preset, without drawing it: what the curve costs at
 * each place, where the knob stands, and the settings a place produces.
 *
 * The application works a curve out over the whole range in one pass, so moving
 * the control computes nothing. The line drawn while that answer is on its way
 * is arithmetic over the settings alone, and is shown as an approximation.
 */
import { BOUNDS, DEFAULTS, NOWHERE } from './core'
import type { Curve, Goal, Place, Point, Rule, Settings } from './core'

/** How many places the line drawn in the answer's place is worked out at. */
const PLACES = 25

/** The shortest day a curve of minutes runs to. */
const LEAST_CEILING = 60

/** How far each end of the retention range stands. */
const RETENTION = BOUNDS.retention

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
export const steers = (goal: Goal): Field => {
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
  const own = new Set<Field>(steers(settings.goal) === 'byDate' ? [] : ['byDate'])
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
 * The budget each goal's own value closes a day by, as the preset writes the
 * key. A target closes no day of its own: what closes a day worked to one is
 * always a count.
 */
const CLOSES: Record<Goal, readonly string[]> = {
  minutes: ['minutes_a_day'],
  retention: [],
  date: ['by_date'],
}

/** Every budget a preset closes a day by, which is the whole of what may be said. */
const CLOSERS: readonly string[] = ['minutes_a_day', 'new_a_day', 'reviews_a_day']

/**
 * What closes the day here besides the goal on screen, and nothing where the
 * goal is the whole of it. A day nothing closed asked for every card there was,
 * and a day two budgets closed names both.
 */
export const limiting = (curve: Curve, point: Point | null): readonly string[] => {
  if (!curve.honest || !point) return []
  return point.closed.filter((one) => CLOSERS.includes(one) && !CLOSES[curve.goal].includes(one))
}

/**
 * The day the overdue pile is gone, read off the very projection the band is
 * drawn from. Null is a place with nothing overdue to be gone at all, and -1
 * is a pile still standing on the last day projected.
 */
export const clearing = (backlog: readonly number[]): number | null => {
  if (!backlog.some((one) => one > 0)) return null
  const at = backlog.indexOf(0)
  return at < 0 ? -1 : at + 1
}

/** A number held inside the bounds of the setting it is. */
export const held = (value: number, of: keyof typeof BOUNDS): number =>
  Math.min(Math.max(value, BOUNDS[of].least), BOUNDS[of].most)

/** The place of the grid nearest a value, and the last one for an empty grid. */
export const nearest = (grid: readonly number[], value: number): number => {
  if (grid.length === 0) return -1
  let at = 0
  for (let i = 1; i < grid.length; i += 1) {
    if (Math.abs((grid[i] ?? 0) - value) < Math.abs((grid[at] ?? 0) - value)) at = i
  }
  return at
}

/** The place a fraction of the way along a grid, held inside it. */
export const placeAt = (grid: readonly number[], share: number): number => {
  if (grid.length === 0) return -1
  const last = grid.length - 1
  return Math.min(Math.max(Math.round(share * last), 0), last)
}

/** The day a number of days from another, written as the year, month and day. */
export const dayAfter = (from: Date, days: number): string => {
  const day = new Date(Date.UTC(from.getUTCFullYear(), from.getUTCMonth(), from.getUTCDate()))
  day.setUTCDate(day.getUTCDate() + days)
  return day.toISOString().slice(0, 10)
}

/** Whether a value is a day at all, which an empty field is not. */
export const isDay = (day: string): boolean => !Number.isNaN(Date.parse(`${day}T00:00:00Z`))

/** How many days stand between today and a day, and zero for a day that is not one. */
export const daysUntil = (today: Date, day: string): number => {
  const named = Date.parse(`${day}T00:00:00Z`)
  if (Number.isNaN(named)) return 0
  const from = Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate())
  return Math.round((named - from) / 86400000)
}

/** Why a preset's goal has nothing to work on, and empty where it has. */
export type Idle = 'unpointed' | 'noCards' | 'beginsNothing' | ''

/**
 * Whether the goal has nothing to work on, and why. The counts the curve
 * carries say the first two: no deck points here, or the decks that do hold
 * nothing between them. A preset holding cards is never told it holds none.
 *
 * The third is a preset that schedules and has nothing it can schedule: every
 * card face here is one nobody has begun, and no place of the range begins one.
 * It is a fact about the material, and not a reason the preset is stopped.
 */
export const idle = (curve: Curve): Idle => {
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
 */
export const approximate = (settings: Settings, today: Date): Curve => {
  const grid = gridFor(settings, today)
  const at = grid.map((value) => guessed(settings, value, grid))
  const days = settings.goal === 'date' ? grid.map((value) => dayAfter(today, value)) : []
  const value = standing(settings, today)
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

/** Where the preset itself stands, in the units of its goal's grid. */
export const standing = (settings: Settings, today: Date): number => {
  if (settings.goal === 'retention') return settings.retention
  if (settings.goal === 'date') return daysUntil(today, settings.byDate)
  return settings.minutesADay
}

/** The whole range of a goal, at the places the line is drawn at. */
const gridFor = (settings: Settings, today: Date): readonly number[] => {
  if (settings.goal === 'retention') {
    return ladder(RETENTION.least, RETENTION.most, (one) => Math.round(one * 1000) / 1000)
  }
  if (settings.goal === 'date') {
    const most = Math.max(daysUntil(today, settings.byDate), 30)
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
export const producing = (
  was: Settings,
  place: number,
  curve: Curve,
  today: Date,
): Settings => {
  const value = curve.grid[place] ?? standing(was, today)
  if (curve.goal === 'retention') {
    return { ...was, retention: held(round(value, 2), 'retention') }
  }
  if (curve.goal === 'date') {
    return { ...was, byDate: curve.days[place] ?? dayAfter(today, value) }
  }
  return { ...was, minutesADay: held(Math.round(value), 'minutesADay') }
}

/** A number to that many places. */
export const round = (value: number, places: number): number => {
  const scale = 10 ** places
  return Math.round(value * scale) / scale
}
