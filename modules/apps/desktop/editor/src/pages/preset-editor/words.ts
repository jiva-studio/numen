// Stryker disable StringLiteral: interface copy
import type { StopReason } from '@numen/protocol'
import { formatStopReasonMessage } from '@numen/wire'
/** What a preset tab says: the one control, the settings under it, and what went wrong. */
import { many, percent, plural } from '@numen/ui'
import type { ErrorCode } from '@/shared/errors'
import { getErrorWords } from '@/shared/words/errors'
import type { BudgetUnit, Goal, Rule } from './types'
import type { Field } from './lib/fields'
import {
  getBuyFacts,
  getLearningFacts,
  getMaterialFacts,
  type BuyFact,
  type BuyInput,
  type LearningFact,
  type MaterialFact,
} from './lib/facts'

/** What each of the three goals is offered as: the value it steers. */
const GOALS: Record<Goal, string> = {
  minutes: 'Minutes a day',
  retention: 'Retention',
  date: 'A date',
}

/**
 * What each of the two rules for the learned is offered as: a name, since a
 * control offers names and the row under it says what they mean.
 */
const RULES: Record<Rule, string> = {
  interval: 'By days',
  retention: 'By remembering',
}

/** What each of the two units a budget is spent in is offered as. */
const BUDGET_UNITS: Record<BudgetUnit, string> = {
  cards: 'Cards',
  shows: 'Showings',
}

/** What each setting is called, and the one line that says what it is. */
const FIELDS: Record<Field, readonly [string, string]> = {
  newADay: ['New a day', 'How many unseen cards a day holds.'],
  reviewsADay: ['Reviews a day', 'How many returning cards a day holds.'],
  retention: ['Retention', 'How much of what you are asked you would remember.'],
  minutesADay: [
    'Minutes a day',
    'How long a day of review runs, spent against the time each answer took.',
  ],
  byDate: ['The day', 'The day the material is to be in the head.'],
  counts: ['Counts', "What a day's budget is spent on."],
  learned: [
    'Counts as learned',
    'By days, a card is learned once its reviews stand at least so many days apart. ' +
      'By remembering, once the chance you would remember it today is at least the target.',
  ],
  interval: ['Days apart', 'How far apart reviews stand before a card counts as learned.'],
  backlog: [
    'Overdue share',
    'What part of a session goes to the overdue pile before new material is offered, in per cent. ' +
      'It moves what fills a day and not how much it holds, and at nothing the new material ' +
      'goes first while there is any.',
  ],
  load: [
    'Load by day',
    'What part of a day of review each day of the week carries, in per cent. ' +
      'A day at nothing admits nothing, and a card falling there stands overdue until a day ' +
      'that admits it; an even load leans a card off a lighter day where its own interval ' +
      'leaves room to lean, and nothing holds a day to its share.',
  ],
  hasEvenLoad: [
    'Even load',
    'Whether a card is leaned towards a lighter day of the days its own interval allows. ' +
      'A card whose interval allows no other day stays where it fell.',
  ],
}

/** An error clause as a sentence. */
const sayAsSentence = (clause: string): string =>
  `${clause.charAt(0).toUpperCase()}${clause.slice(1)}.`

/** A count as a person reads one. */
const count = (value: number): string => `${Math.round(value)}`

/** A fact about what a place buys, said. */
const formatBuyFact = (fact: BuyFact): string => {
  switch (fact.kind) {
    case 'daysOff':
      return `${many(fact.days, 'day')} off`
    case 'minutesADay':
      return `${many(fact.minutes, 'minute')} a day`
    case 'remembered':
      return `${percent(fact.share)} remembered`
    case 'cardsASession':
      return `${many(fact.cards, 'card')} a session`
    case 'cannotGetThere':
      return `${count(fact.short)} of ${count(fact.cards)} cannot get there`
    case 'overdueGone':
      return `overdue gone in ${many(fact.days, 'day')}`
    case 'overdueStays':
      return `not within ${many(fact.horizon, 'day')}`
  }
}

/** A fact about the material, said as a figure and the name under it. */
const formatMaterialFact = (fact: MaterialFact): { figure: string; name: string } => {
  const figure = count(fact.count)
  switch (fact.kind) {
    case 'decks':
      return { figure, name: plural(fact.count, 'deck') }
    case 'cards':
      return { figure, name: plural(fact.count, 'card') }
    case 'overdue':
      return { figure, name: 'overdue' }
    case 'unbegun':
      return { figure, name: 'new' }
  }
}

/** A fact about what is learned, said as a figure and the name under it. */
const formatLearningFact = (fact: LearningFact): { figure: string; name: string } => {
  switch (fact.kind) {
    case 'notYet':
      return { figure: 'not yet', name: 'in the days ahead' }
    case 'allToday':
      return { figure: 'today', name: 'all of it learned' }
    case 'daysToLearn':
      return {
        figure: count(fact.days),
        name: plural(fact.days, 'day to learn it', 'days to learn it'),
      }
    case 'learnedToday':
      return { figure: `${count(fact.learned)} of ${count(fact.cards)}`, name: 'learned today' }
  }
}

export const WORDS = {
  preset: 'Preset',
  /** What a preset tab is called before the vault has said what the note is. */
  newPreset: 'Preset',
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
  buys: (goal: Goal, at: BuyInput): readonly string[] => getBuyFacts(goal, at).map(formatBuyFact),
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
  material: (decks: number, cards: number, overdue: number, fresh: number) =>
    getMaterialFacts(decks, cards, overdue, fresh).map(formatMaterialFact),
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
  /** The value the control stands at, in the units of its goal. */
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
  /** What a write was refused for, and where the settings stand after it. */
  notSaved: (error: ErrorCode) =>
    `${sayAsSentence(getErrorWords(error))} These settings are still here.`,
  /** The vault answered a read with neither settings nor a reason. */
  unreachable: 'The vault would not answer for this preset, and did not say why.',
  /** The vault answered a write with neither a file nor a reason. */
  unwritten: 'The vault would not take these settings, and did not say why. They are still here.',
  /** The vault would not work the picture out. */
  noCurve: 'The vault would not work this picture out, and did not say why.',
}
