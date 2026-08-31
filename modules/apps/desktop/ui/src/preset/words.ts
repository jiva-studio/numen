/** What a preset tab says: the one control, the settings under it, and what went wrong. */
import type { Counts, Goal } from './core'
import type { Field } from './curve'

/** What each of the three goals is offered as: the value it steers. */
const GOALS: Record<Goal, string> = {
  minutes: 'Minutes a day',
  retention: 'Retention',
  date: 'A date',
}

/** What each of the two things a budget is spent on is offered as. */
const COUNTS: Record<Counts, string> = {
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
  lightDays: ['Light days', 'The days of the week the load is cut on.'],
  evenLoad: ['Even load', 'Whether days are made to resemble each other.'],
}

/** A share as a person reads one, which is a percentage and not a fraction. */
const share = (value: number): string => `${Math.round(value * 100)}%`

/** A count as a person reads one. */
const count = (value: number): string => `${Math.round(value)}`

/** A count and the thing it counts, in the singular where there is one of it. */
const many = (value: number, one: string, more = `${one}s`): string =>
  `${count(value)} ${Math.round(value) === 1 ? one : more}`

export const WORDS = {
  preset: 'Preset',
  /** What a preset tab is called before the vault has said what the note is. */
  newPreset: 'Preset',
  /** What the three segments are, said over them. */
  goal: 'Goal',
  goalName: (goal: Goal) => GOALS[goal],
  countsName: (counts: Counts) => COUNTS[counts],
  /** What each axis measures, said along the axis it names. */
  axisY: (goal: Goal) => (goal === 'minutes' ? 'Cards in a sitting' : 'Minutes a day'),
  axisX: (goal: Goal) => {
    if (goal === 'retention') return 'Retention'
    return goal === 'date' ? 'Days from today' : 'Minutes a day'
  },
  /** One height of the picture, against the line it is the height of. */
  heightAt: (goal: Goal, value: number) =>
    goal === 'minutes' ? many(value, 'card') : `${count(value)} min`,
  /** One place along the picture, in the units of the goal's grid. */
  widthAt: (goal: Goal, value: number) => {
    if (goal === 'retention') return share(value)
    return goal === 'date' ? `${count(value)} d` : `${count(value)} min`
  },
  /**
   * How far behind the preset stands. Overdue is the backlog alone — a card
   * whose day came and went — and never what a sitting puts in front of a
   * person, which is that backlog and today's cards together.
   */
  behind: (overdue: number, cards: number) =>
    overdue === 1
      ? `1 of ${many(cards, 'card face')} is overdue.`
      : `${count(overdue)} of ${many(cards, 'card face')} are overdue.`,
  /** How long that backlog takes to clear at the place the knob stands at. */
  clearing: (days: number) =>
    days < 0
      ? 'At this pace the backlog never clears.'
      : `At this pace nothing is overdue after ${many(days, 'day')}.`,
  /** A day the card limits close before its minutes run out. */
  closed: (newADay: number, reviewsADay: number) =>
    `A longer day buys nothing here: ${count(newADay)} new and ` +
    `${many(reviewsADay, 'review')} a day close the day before its minutes run out.`,
  fieldName: (field: Field) => FIELDS[field][0],
  fieldDetail: (field: Field) => FIELDS[field][1],
  settings: 'What the goal produced',
  /** The knob, and what it is announced as while it is moved. */
  knob: 'The goal of this preset',
  /** Said in the picture's place while the application works the curve out. */
  waiting: 'Reading the vault…',
  /** The three marks on the curve, each named where it stands. */
  now: 'you are here',
  /**
   * The other mark, named for what it is under each goal. Under minutes it is
   * where the clock stops being the limit, which is not a recommendation.
   */
  markName: (goal: Goal) => {
    if (goal === 'retention') return 'most kept'
    return goal === 'date' ? 'first day it fits' : 'time enough'
  },
  /** The rule that mark is found by, said under the picture and not on it. */
  markRule: (goal: Goal) => {
    if (goal === 'retention') {
      return 'Most kept: the target that leaves most of the material in the head.'
    }
    if (goal === 'date') {
      return 'First day it fits: the first day the budget this preset keeps gets through the material.'
    }
    return (
      'Time enough: the shortest day the clock no longer cuts short. ' +
      'Below it the day ends before the material does; above it a longer day adds nothing.'
    )
  },
  /** The figure is the window's own arithmetic, and the answer is on its way. */
  about: 'about',
  aboutMeaning: 'a figure the window guessed while the application works out the honest one',
  /** The value the control stands at, in the units of its goal. */
  value: (goal: Goal, value: number, day: string) => {
    if (goal === 'retention') return `${share(value)} remembered`
    if (goal === 'date') return `${many(value, 'day')} to ${day}`
    return `${many(value, 'minute')} a day`
  },
  /** What standing there costs, in one sentence, in the words of the goal chosen. */
  costs: (goal: Goal, value: number, reviews: number, minutes: number, retained: number) => {
    if (goal === 'retention') {
      return (
        `Remembering ${share(value)} of what you are asked is ` +
        `${many(minutes, 'minute')} a day and ${many(reviews, 'card')}.`
      )
    }
    if (goal === 'date') return `Being through it by then is ${many(minutes, 'minute')} a day.`
    return (
      `A sitting of ${many(value, 'minute')} puts ${many(reviews, 'card')} in front of you, ` +
      `and you would remember ${share(retained)} of what you are asked.`
    )
  },
  /** The arithmetic under a goal of a date, with the sum already done. */
  owing: (cards: number) => `${many(cards, 'card')} would still be owed on that day.`,
  needing: (needed: number, standing: number) =>
    `This preset keeps ${many(standing, 'minute')} a day, ` +
    `and getting through it by then takes ${many(needed, 'minute')}.`,
  through: (part: number, standing: number) =>
    `At ${many(standing, 'minute')} a day, ${count(part * 100)}% of it is through by then.`,
  /** No budget at all gets through the material by the day named. */
  unmet: 'No day of review gets through the material by that day.',
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
  /** The preset schedules nothing, for either of the two reasons. */
  spent: 'This preset is past the day it aimed at. Its budget is spent, and it schedules nothing.',
  paused: 'No cards a day: this preset schedules nothing, and every deck pointing at it stops.',
  /** What is wrong with the file, said above the control. */
  problems: 'What is wrong with this preset',
  /** The file moved under the window, and the two answers to that. */
  changed: 'This file changed on disk, so nothing was written.',
  reads: 'Read it again',
  /** The write and the read were refused, and the vault said nothing at all. */
  notSaved: 'These settings could not be written.',
  refused: 'This preset could not be read.',
  unreachable: 'The vault could not be reached.',
  notAPreset: 'That note is not a preset, and the defaults stand.',
}
