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
  openPreset: 'Open the preset',
  /** What a preset tab is called before the vault has said what the note is. */
  newPreset: 'Preset',
  /** The deck in front names no preset, so the defaults schedule it. */
  noPreset: 'This deck names no preset, so it is scheduled by the defaults.',
  /** What the three segments are, said over them. */
  goal: 'Goal',
  goalName: (goal: Goal) => GOALS[goal],
  budgetUnitName: (unit: BudgetUnit) => BUDGET_UNITS[unit],
  ruleName: (rule: Rule) => RULES[rule],
  /** What each axis measures, said along the axis it names. */
  axisY: (goal: Goal) => (goal === 'minutes' ? 'Cards in a session' : 'Minutes a day'),
  axisX: (goal: Goal) => {
    if (goal === 'retention') return 'Retention'
    return goal === 'date' ? 'Days from today' : 'Minutes a day'
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
  settings: 'What the goal produced',
  /** The knob, and what it is announced as while it is moved. */
  knob: 'The goal of this preset',
  /** Said in the picture's place while the application works the curve out. */
  waiting: 'Loading…',
  /**
   * The suggested mark, named for what it is under each goal. Under minutes it
   * is where the clock stops being the limit.
   */
  markName: (goal: Goal) => {
    if (goal === 'retention') return 'most kept'
    return goal === 'date' ? 'first day it fits' : 'time enough'
  },
  value: (goal: Goal, value: number, day: string) => {
    if (goal === 'retention') return `${percent(value)} remembered`
    if (goal === 'date') return `${many(value, 'day')} to ${day}`
    return `${many(value, 'minute')} a day`
  },
  /** No deck points here, so the goal has nothing to work on. */
  unpointed:
    'No deck points at this preset, so it schedules nothing. ' +
    'Point a deck at it and its goal has cards to work on.',
  /** The decks pointing here hold no cards between them. */
  noCards: (decks: number) =>
    decks === 1
      ? 'One deck points here and it holds no cards, so this preset schedules nothing.'
      : `${count(decks)} decks point here and they hold no cards, ` +
        'so this preset schedules nothing.',
  /** The decks hold cards, and none of them is one this preset can begin. */
  beginsNothing:
    'Nobody has begun a card in these decks and this preset begins none a day, ' +
    'so it has nothing to schedule. Raise the new cards a day and its goal has ' +
    'cards to work on.',
  /** Why the preset schedules nothing on the day it was read in. */
  stopped: (why: StopReason) => {
    const clause = formatStopReasonMessage(why)
    return clause ? `This preset schedules nothing: ${clause}.` : ''
  },
  /** What is wrong with the file, said above the control. */
  problems: 'What is wrong with this preset',
  /** The file moved under the window, and the two answers to that. */
  changed: 'This file changed on disk, so nothing was written.',
  reads: 'Read it again',
  /** What a read failed for. */
  notRead: (error: ErrorCode) => sayAsSentence(getErrorWords(error)),
  notSaved: (error: ErrorCode) =>
    `${sayAsSentence(getErrorWords(error))} These settings are still here.`,
  /** The vault answered a read with neither settings nor a reason. */
  unreachable: 'The vault would not answer for this preset, and did not say why.',
  /** The vault answered a write with neither a file nor a reason. */
  unwritten: 'The vault would not take these settings, and did not say why. They are still here.',
  /** The vault would not work the picture out. */
  noCurve: 'The vault would not work this picture out, and did not say why.',
}
