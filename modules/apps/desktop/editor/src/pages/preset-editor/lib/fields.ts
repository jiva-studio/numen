/**
 * Which settings a goal schedules by, and the one word that says their shape.
 *
 * A goal names one budget, and the settings of the other two take no part in
 * it: they are neither drawn nor written while it stands, and the file keeps
 * them where the person left them.
 */
import type { Goal, Rule, Settings } from '../types'

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
export const fieldsUnder = (goal: Goal, rule: Rule): readonly Field[] => {
  const drawn = new Set<Field>([
    ...BUDGETS[goal],
    ...SPENDING[goal],
    ...LEARNS[rule],
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
