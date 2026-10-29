// Stryker disable StringLiteral: interface copy
import type { StopReason } from '../api/types.js'
import { type ErrorCode, getErrorWords } from '../cards/wire/errors'
import { formatStopReasonMessage } from '../cards/wire/stop'
/** Interface copy of the preset tab: the control, its settings, and error messages. */
import { many, percent } from '@numen/ui'
import type { BudgetUnit, Goal, Rule } from './types'
import type { Field } from './lib/fields'
import {
  getBuyFacts,
  getLearningFacts,
  getMaterialFacts,
  type BuyInput,
  type MaterialCounts,
} from './lib/facts'
import { BUDGET_UNITS, FIELDS, GOALS, RULES } from './lib/labels'
import {
  count,
  formatBuyFact,
  formatLearningFact,
  formatMaterialFact,
  sayAsSentence,
} from './lib/formatFacts'

export const WORDS = {
  preset: 'Preset',
  /** The command that opens the preset of the note in front. */
  openPreset: 'Open preset tab',
  /** Tab title used until the vault supplies the note title. */
  newPreset: 'Preset',
  /** The deck in front names no preset, so the defaults schedule it. */
  noPreset: 'No preset set. Scheduled by the defaults.',
  /** Labels of the three goal segments, shown above them. */
  goal: 'Goal',
  goalName: (goal: Goal) => GOALS[goal],
  budgetUnitName: (unit: BudgetUnit) => BUDGET_UNITS[unit],
  ruleName: (rule: Rule) => RULES[rule],
  /** Axis titles of the curve, one per axis. */
  axisY: (goal: Goal) => (goal === 'minutes' ? 'Cards in a session' : 'Minutes per day'),
  axisX: (goal: Goal) => {
    if (goal === 'retention') return 'Retention'
    return goal === 'date' ? 'Days from today' : 'Minutes per day'
  },
  /** Axis titles of the backlog chart. */
  backlogY: 'Cards overdue',
  backlogX: 'Days from today',
  /** One height of the backlog, and one day along it. */
  formatBacklogCards: (value: number) => many(value, 'card'),
  formatBacklogDays: (days: number) => `${count(days)} d`,
  /**
   * Lines of the bubble above the knob: the current value, and below it what that
   * value buys. Each line is short enough to read at a glance.
   *
   * A goal of a date has no overdue line: every card is scheduled by that day, so
   * the day the backlog clears carries no information. No overdue line appears when
   * nothing is overdue.
   */
  buys: (goal: Goal, input: BuyInput): readonly string[] =>
    getBuyFacts(goal, input).map(formatBuyFact),
  /** Formats one vertical-axis value of the curve in the unit of the goal. */
  formatGoalCost: (goal: Goal, value: number) =>
    goal === 'minutes' ? many(value, 'card') : `${count(value)} min`,
  /** Formats one horizontal-axis value of the curve in the units of the goal's grid. */
  formatGoalValue: (goal: Goal, value: number) => {
    if (goal === 'retention') return percent(value)
    return goal === 'date' ? `${count(value)} d` : `${count(value)} min`
  },
  /**
   * Lines describing what the control acts on: the decks that point at the preset
   * and the material they hold in total.
   */
  material: (counts: MaterialCounts) => getMaterialFacts(counts).map(formatMaterialFact),
  /**
   * Lines stating when the material is learned and how much is learned now, taken
   * from the run the curve is drawn from. When no day exists on which the whole
   * material is learned, no day is stated and the line gives the count learned
   * today. A pace whose days run out with a card still unlearned states that the
   * learned day is not reached.
   */
  learning: (learns: number | undefined, learned: number, cards: number) =>
    getLearningFacts(learns, learned, cards).map(formatLearningFact),
  fieldName: (field: Field) => FIELDS[field][0],
  fieldDetail: (field: Field) => FIELDS[field][1],
  /** A share in hundredths, read out beside the track it is moved along. */
  percent: (value: number) => `${count(value)}%`,
  settings: 'Resulting settings',
  /** The knob, and what it is announced as while it is moved. */
  knob: 'Preset goal',
  /** Placeholder shown in place of the curve while the application computes it. */
  waiting: 'Loading…',
  /**
   * The suggested mark, named for what it is under each goal. Under minutes it
   * is where the clock stops being the limit.
   */
  markName: (goal: Goal) => {
    if (goal === 'retention') return 'best retention'
    return goal === 'date' ? 'earliest date' : 'enough time'
  },
  value: (goal: Goal, value: number, day: string) => {
    if (goal === 'retention') return `${percent(value)} remembered`
    if (goal === 'date') return `${many(value, 'day')} to ${day}`
    return `${many(value, 'minute')} a day`
  },
  /** No deck points here, so the goal has nothing to work on. */
  unpointed: 'No decks use this preset. Set it on a deck to schedule cards.',
  /** The decks pointing here hold no cards between them. */
  noCards: (decks: number) =>
    decks === 1
      ? 'The deck using this preset has no cards.'
      : `The ${count(decks)} decks using this preset have no cards.`,
  /** The decks hold cards, and none of them is one this preset can begin. */
  beginsNothing:
    'No cards started and new cards per day is 0. Raise new cards per day to schedule cards.',
  /** Why the preset schedules nothing on the day it was read in. */
  stopped: (why: StopReason) => {
    const clause = formatStopReasonMessage(why)
    return clause ? `No cards scheduled: ${clause}.` : ''
  },
  /** Heading of the list of problems with the file, shown above the control. */
  problems: 'Preset problems',
  /** The file moved under the window, and the two answers to that. */
  changed: 'Not saved: the file changed on disk.',
  reads: 'Reload',
  /** What a read failed for. */
  notRead: (error: ErrorCode) => sayAsSentence(getErrorWords(error)),
  notSaved: (error: ErrorCode) => `${sayAsSentence(getErrorWords(error))} Your settings are kept.`,
  /** The vault answered a read with neither settings nor a reason. */
  unreachable: 'Preset unreadable: no response from the vault.',
  /** The vault answered a write with neither a file nor a reason. */
  unwritten: 'Settings not saved: no response from the vault. Your settings are kept.',
  /** The vault could not compute the curve. */
  noCurve: 'Chart unavailable: no response from the vault.',
}
