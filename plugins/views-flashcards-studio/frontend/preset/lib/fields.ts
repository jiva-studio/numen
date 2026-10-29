/**
 * The settings a goal schedules by, and the single value that names their set.
 *
 * A goal selects one budget, and the settings of the other two budgets do not
 * apply to it: they are neither displayed nor written while the goal is active,
 * and the file keeps their stored values.
 */
import type { Bounds, Goal, Rule, Settings, SettingsBounds } from '../types'
import { round } from './curve'

/** One setting of a preset that the user may set directly. */
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
  | 'hasEvenLoad'

/** Every setting the receipt has a row for, in the order they stand in. */
export const FIELDS: readonly Field[] = [
  'newADay',
  'reviewsADay',
  'learned',
  'interval',
  'retention',
  'minutesADay',
  'byDate',
  'counts',
  'backlog',
  'load',
  'hasEvenLoad',
]

/**
 * The budget each goal schedules by.
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
 * The share of a day spent on overdue cards before any new card is offered.
 * It allocates the day's budget and does not close the day, so it is listed
 * beside a budget and not among the budgets. A goal of a date covers the whole
 * material by its own schedule and has no backlog share.
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
const ALWAYS: readonly Field[] = ['load', 'hasEvenLoad']

/** The settings a goal schedules by, which are the rows the receipt draws. */
export const getGoalFields = (goal: Goal, rule: Rule): readonly Field[] => {
  const goalFields = new Set<Field>([
    ...BUDGETS[goal],
    ...SPENDING[goal],
    ...LEARNS[rule],
    ...ALWAYS,
  ])
  return FIELDS.filter((field) => goalFields.has(field))
}

/** The field the goal steers, which is the knob under another name. */
export const getGoalField = (goal: Goal): Field => {
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
export const getShapeKey = (settings: Settings): string => {
  // Stryker disable next-line ArrayDeclaration: Goal field byDate excludes no fields from shape key.
  const own = new Set<Field>(getGoalField(settings.goal) === 'byDate' ? [] : ['byDate'])
  const values: Record<Field, string> = {
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
      .sort((a, b) => a.localeCompare(b))
      .map((day) => `${day}=${settings.load[day]}`)
      .join(','),
    hasEvenLoad: `${settings.hasEvenLoad}`,
  }
  const rest = FIELDS.filter((field) => !own.has(field)).map((field) => `${field}=${values[field]}`)
  return [settings.goal, ...rest].join(' ')
}

export type FieldLimits = { min: number; max: number } | Record<string, never>

/** Retention is held as a share of one and shown as a percentage. */
const PERCENT = 100

const toLimits = (bounds: Bounds | undefined, per = 1): FieldLimits =>
  bounds ? { min: bounds.min * per, max: bounds.max * per } : {}

export const getFieldLimits = (field: Field, bounds: SettingsBounds): FieldLimits => {
  if (field === 'newADay') return toLimits(bounds.newADay)
  if (field === 'reviewsADay') return toLimits(bounds.reviewsADay)
  if (field === 'minutesADay') return toLimits(bounds.minutesADay)
  if (field === 'backlog') return toLimits(bounds.backlog)
  if (field === 'interval') return toLimits(bounds.interval)
  if (field === 'retention') return toLimits(bounds.retention, PERCENT)
  return {}
}

/** The number a field's number control shows, and nothing for a field that has none. */
export const getFieldCount = (field: Field, settings: Settings): number | null => {
  if (field === 'newADay') return settings.newADay
  if (field === 'reviewsADay') return settings.reviewsADay
  if (field === 'retention') return Math.round(settings.retention * PERCENT)
  if (field === 'minutesADay') return settings.minutesADay
  if (field === 'interval') return settings.interval
  return null
}

/** The value a field is set to by the number its control shows. */
export const getFieldValue = (field: Field, count: number): number =>
  field === 'retention' ? round(count / PERCENT, 2) : count
