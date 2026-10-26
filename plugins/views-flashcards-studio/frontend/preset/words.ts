// Stryker disable StringLiteral: interface copy
import type { StopReason } from '../gen/numen/flashcards/v1/presets_pb.js'
import { type ErrorCode, getErrorWords } from '../cards/wire/errors'
import { formatStopReasonMessage } from '../cards/wire/stop'
/** What a preset tab says: the one control, the settings under it, and what went wrong. */
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
  openPreset: 'Open preset',
  /** What a preset tab is called before the vault has said what the note is. */
  newPreset: 'Preset',
  /** The deck in front names no preset, so the defaults schedule it. */
  noPreset: 'No preset set. Scheduled by the defaults.',
  /** What the three segments are, said over them. */
  goal: 'Goal',
  goalName: (goal: Goal) => GOALS[goal],
  budgetUnitName: (unit: BudgetUnit) => BUDGET_UNITS[unit],
  ruleName: (rule: Rule) => RULES[rule],
  /** What each axis measures, said along the axis it names. */
  axisY: (goal: Goal) => (goal === 'minutes' ? 'Cards in a session' : 'Minutes per day'),
  axisX: (goal: Goal) => {
    if (goal === 'retention') return 'Retention'
    return goal === 'date' ? 'Days from today' : 'Minutes per day'
  },
  /** What the backlog under the picture measures, said along the axis it names. */
  backlogY: 'Cards overdue',
  backlogX: 'Days from today',
  /** One height of the backlog, and one day along it. */
  formatBacklogCards: (value: number) => many(value, 'card'),
  formatBacklogDays: (days: number) => `${count(days)} d`,
  /**
   * What the bubble over the knob says: the value being held, and under it what
   * that value buys. Each line is short enough to be taken at a glance.
   *
   * A goal of a date says nothing about the overdue, since every card is to be
   * got through by that day and the day the pile goes says nothing. Where
   * nothing is overdue there is no such line either.
   */
  buys: (goal: Goal, input: BuyInput): readonly string[] =>
    getBuyFacts(goal, input).map(formatBuyFact),
  /** One height of the picture, against the line it is the height of. */
  formatGoalCost: (goal: Goal, value: number) =>
    goal === 'minutes' ? many(value, 'card') : `${count(value)} min`,
  /** One place along the picture, in the units of the goal's grid. */
  formatGoalValue: (goal: Goal, value: number) => {
    if (goal === 'retention') return percent(value)
    return goal === 'date' ? `${count(value)} d` : `${count(value)} min`
  },
  /**
   * What the control is acting on, said over the picture: the decks that point
   * here and the material they hold between them.
   */
  material: (counts: MaterialCounts) => getMaterialFacts(counts).map(formatMaterialFact),
  /**
   * When the material is learned, and how much of it stands learned now, read
   * off the run the picture is drawn from. Where there is no day the whole of
   * it stands learned on, nothing is said in its place: what is left is what
   * stands learned today. A pace whose days run out with one still to learn
   * has a day to speak of and says it has not come.
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
  /** Said in the picture's place while the application works the curve out. */
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
  /** What is wrong with the file, said above the control. */
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
  /** The vault would not work the picture out. */
  noCurve: 'Chart unavailable: no response from the vault.',
}
