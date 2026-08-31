/**
 * The one control of a preset, without drawing it: what the curve costs at
 * each place, where the knob stands, and the settings a place produces.
 *
 * The application works a curve out over the whole range in one pass, so
 * moving the control computes nothing. What is here besides is the line the
 * window draws in its place while that answer is on its way, which is arithmetic
 * over the settings alone and is shown as the approximation it is.
 */
import { BOUNDS, DEFAULTS, NOWHERE } from './core'
import type { Curve, Goal, Mark, Point, Settings } from './core'

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
  | 'byDate'
  | 'counts'
  | 'lightDays'
  | 'evenLoad'

/** Every setting the receipt has a row for, in the order they stand in. */
export const FIELDS: readonly Field[] = [
  'newADay',
  'reviewsADay',
  'retention',
  'minutesADay',
  'byDate',
  'counts',
  'lightDays',
  'evenLoad',
]

/**
 * The settings a goal schedules by, which are the rows the receipt draws. A
 * day steers nothing unless it is the goal, so it stands under that goal alone.
 */
export const fieldsUnder = (goal: Goal): readonly Field[] =>
  FIELDS.filter((field) => field !== 'byDate' || goal === 'date')

/** The fields a goal fills in itself. The value the goal names is its own. */
export const producedBy = (goal: Goal): readonly Field[] =>
  goal === 'minutes' ? ['reviewsADay', 'retention'] : ['minutesADay', 'reviewsADay']

/** The field the goal steers, which is the knob under another name. */
export const steers = (goal: Goal): Field => {
  if (goal === 'minutes') return 'minutesADay'
  if (goal === 'retention') return 'retention'
  return 'byDate'
}

/**
 * The settings that give a curve its shape, as one word. The value the knob
 * rides and the values it produces are left out, so walking the grid reads the
 * same shape throughout and a curve is asked for once for it.
 */
export const shapeOf = (settings: Settings): string => {
  const own = new Set<Field>([steers(settings.goal), ...producedBy(settings.goal)])
  const said: Record<Field, string> = {
    newADay: `${settings.newADay}`,
    reviewsADay: `${settings.reviewsADay}`,
    retention: `${settings.retention}`,
    minutesADay: `${settings.minutesADay}`,
    byDate: settings.byDate,
    counts: settings.counts,
    lightDays: settings.lightDays.join(','),
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
 * Whether a longer day buys nothing over the whole range, which is a day the
 * card limits close before its minutes run out.
 */
export const closed = (curve: Curve): boolean => {
  if (!curve.honest || curve.goal !== 'minutes' || curve.at.length < 2) return false
  const cards = curve.at.map((point) => costOf(curve.goal, point))
  return Math.max(...cards) - Math.min(...cards) < 1
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
export type Idle = 'unpointed' | 'noCards' | ''

/**
 * Whether the goal has nothing to work on, and why. The two counts the curve
 * carries say it: no deck points here, or the decks that do hold nothing
 * between them. What the curve comes to says nothing about it, so a preset
 * holding cards is never told it holds none.
 */
export const idle = (curve: Curve): Idle => {
  if (!curve.honest) return ''
  if (curve.decks === 0) return 'unpointed'
  return curve.cards === 0 ? 'noCards' : ''
}

/**
 * The fields that no longer follow the goal: those the goal fills in itself
 * where what stands in the file differs from what the goal produces for its
 * own stored value. Nothing is remembered between reads — the file carries the
 * goal and the values, and the two either agree or they do not.
 */
export const offGoal = (settings: Settings, curve: Curve, today: Date): ReadonlySet<Field> => {
  const off = new Set<Field>()
  if (!curve.honest || curve.grid.length === 0) return off
  const place = nearest(curve.grid, standing(settings, today))
  const produced = producing(settings, place, curve, today)
  for (const field of producedBy(settings.goal)) {
    if (!agree(settings, produced, field)) off.add(field)
  }
  return off
}

/** Whether one field reads the same in two settings. */
const agree = (one: Settings, two: Settings, field: Field): boolean => {
  if (field === 'retention') return Math.abs(one.retention - two.retention) < 0.005
  if (field === 'minutesADay') return one.minutesADay === two.minutesADay
  if (field === 'reviewsADay') return one.reviewsADay === two.reviewsADay
  if (field === 'newADay') return one.newADay === two.newADay
  return true
}

/** One field of the settings put back to what the goal produces for it. */
export const following = (
  settings: Settings,
  field: Field,
  curve: Curve,
  today: Date,
): Settings => {
  if (curve.grid.length === 0) return settings
  const place = nearest(curve.grid, standing(settings, today))
  const produced = producing(settings, place, curve, today)
  if (field === 'retention') return { ...settings, retention: produced.retention }
  if (field === 'minutesADay') return { ...settings, minutesADay: produced.minutesADay }
  if (field === 'reviewsADay') return { ...settings, reviewsADay: produced.reviewsADay }
  return settings
}

/** Whether the day the goal names is behind us, which spends the budget. */
export const spent = (settings: Settings, today: Date): boolean =>
  settings.goal === 'date' && settings.byDate !== '' && daysUntil(today, settings.byDate) < 0

/** Whether the preset schedules nothing at all. */
export const paused = (settings: Settings, today: Date): boolean =>
  (settings.newADay === 0 && settings.reviewsADay === 0) || spent(settings, today)

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
  const now: Mark = { at: place, value, day: days[place] ?? '' }
  return {
    goal: settings.goal,
    grid,
    days,
    at,
    now,
    suggested: NOWHERE,
    decks: 0,
    cards: 0,
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
  const flat = { retained: 0, owed: 0, through: 0, enough: true, met: true }
  if (settings.goal === 'retention') {
    const minutes = (settings.minutesADay || DEFAULTS.minutesADay) * ((1 - MIDDLE) / (1 - value))
    return { ...flat, minutes, reviews: (minutes * 60) / ANSWER, retained: value }
  }
  if (settings.goal === 'date') {
    const span = Math.max(grid[grid.length - 1] ?? 1, 1)
    const minutes = ((settings.minutesADay || DEFAULTS.minutesADay) * span) / Math.max(value, 1)
    const enough = minutes <= (settings.minutesADay || DEFAULTS.minutesADay)
    const through = Math.min(value / span, 1)
    return { ...flat, minutes, reviews: (minutes * 60) / ANSWER, through, enough, met: true }
  }
  const retained = (CEILING * value) / (value + HALF)
  return { ...flat, minutes: value, reviews: (value * 60) / ANSWER, retained }
}

/**
 * The settings one place of the curve produces. The goal's own value comes off
 * the grid, and the daily limits off what the preset comes to there.
 */
export const producing = (
  was: Settings,
  place: number,
  curve: Curve,
  today: Date,
): Settings => {
  const value = curve.grid[place] ?? standing(was, today)
  const point = curve.at[place]
  if (!point) return was
  const limits = {
    reviewsADay: held(Math.round(point.reviews), 'reviewsADay'),
    minutesADay: held(Math.round(point.minutes), 'minutesADay'),
  }
  if (curve.goal === 'retention') {
    return { ...was, retention: held(round(value, 2), 'retention'), ...limits }
  }
  if (curve.goal === 'date') {
    return { ...was, byDate: curve.days[place] ?? dayAfter(today, value), ...limits }
  }
  return {
    ...was,
    minutesADay: held(Math.round(value), 'minutesADay'),
    reviewsADay: limits.reviewsADay,
    retention: held(round(point.retained, 2), 'retention'),
  }
}

/**
 * The settings the goal produced, with every field a person typed themselves
 * put back as they typed it.
 */
export const kept = (
  produced: Settings,
  byHand: Settings,
  fields: ReadonlySet<Field>,
): Settings => {
  let out = produced
  if (fields.has('minutesADay')) out = { ...out, minutesADay: byHand.minutesADay }
  if (fields.has('newADay')) out = { ...out, newADay: byHand.newADay }
  if (fields.has('reviewsADay')) out = { ...out, reviewsADay: byHand.reviewsADay }
  if (fields.has('retention')) out = { ...out, retention: byHand.retention }
  if (fields.has('lightDays')) out = { ...out, lightDays: byHand.lightDays }
  if (fields.has('evenLoad')) out = { ...out, evenLoad: byHand.evenLoad }
  return out
}

/** A number to that many places. */
export const round = (value: number, places: number): number => {
  const scale = 10 ** places
  return Math.round(value * scale) / scale
}
