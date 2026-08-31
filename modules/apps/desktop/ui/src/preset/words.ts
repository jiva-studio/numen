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

/**
 * The three shares of a day the backlog row offers, each by what it does. Any
 * other share a file carries is offered as the percentage it is.
 */
const BACKLOGS: Record<number, string> = {
  100: 'Overdue first',
  50: 'Split',
  0: 'New first',
}

/** What closes a day, as the preset writes the key, said as a person reads it. */
const CLOSERS: Record<string, string> = {
  minutes_a_day: 'the minutes a day',
  new_a_day: 'the new a day',
  reviews_a_day: 'the reviews a day',
}

/** What each goal's own value is called where something else is doing the limiting. */
const LIMITED: Record<Goal, string> = {
  minutes: 'The length of the day',
  retention: 'The target',
  date: 'The day named',
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
  backlog: ['Backlog', 'How much of a day goes to what is overdue before anything new is offered.'],
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
  backlogName: (share: number) => BACKLOGS[share] ?? `${count(share)}%`,
  /** What each axis measures, said along the axis it names. */
  axisY: (goal: Goal) => (goal === 'minutes' ? 'Cards in a sitting' : 'Minutes a day'),
  axisX: (goal: Goal) => {
    if (goal === 'retention') return 'Retention'
    return goal === 'date' ? 'Days from today' : 'Minutes a day'
  },
  /** What the band under the picture measures, said along the axis it names. */
  backlogY: 'Cards overdue',
  backlogX: 'Days from today',
  /** One height of the band, and one day along it. */
  backlogHeightAt: (value: number) => many(value, 'card'),
  backlogWidthAt: (days: number) => `${count(days)} d`,
  /** One height of the picture, against the line it is the height of. */
  heightAt: (goal: Goal, value: number) =>
    goal === 'minutes' ? many(value, 'card') : `${count(value)} min`,
  /** One place along the picture, in the units of the goal's grid. */
  widthAt: (goal: Goal, value: number) => {
    if (goal === 'retention') return share(value)
    return goal === 'date' ? `${count(value)} d` : `${count(value)} min`
  },
  /**
   * What the control is acting on, said over the picture: the decks that point
   * here and the material they hold between them.
   */
  material: (decks: number, cards: number, overdue: number, fresh: number) => {
    const said = [
      { figure: count(decks), name: decks === 1 ? 'deck' : 'decks' },
      { figure: count(cards), name: cards === 1 ? 'card' : 'cards' },
    ]
    // A figure standing at nothing is left out rather than said as a nought.
    if (overdue > 0) said.push({ figure: count(overdue), name: 'overdue' })
    if (fresh > 0) said.push({ figure: count(fresh), name: 'new' })
    return said
  },
  /**
   * How far behind the preset stands and what this pace does about it, as one
   * sentence. Overdue is the backlog alone — a card whose day came and went —
   * and never what a sitting puts in front of a person, which is that backlog
   * and the cards of the day together.
   */
  behind: (overdue: number, cards: number, days: number) => {
    const said =
      overdue === 1
        ? `1 of ${many(cards, 'card')} is overdue`
        : `${count(overdue)} of ${many(cards, 'card')} are overdue`
    const then =
      days < 0
        ? 'this pace does not get on top of it'
        : `this pace has nothing overdue after ${many(days, 'day')}`
    return `${said}, and ${then}.`
  },
  /**
   * What closes the day where the goal on screen does not, which names the
   * number that would have to move.
   */
  limiting: (goal: Goal, closed: string) =>
    `${LIMITED[goal]} is not what limits this preset here: ${CLOSERS[closed]} ` +
    'closes the day first, and that is the number to move.',
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
  /**
   * What that mark stands at and what the figure means, as one sentence in the
   * paragraph. The value comes first and the meaning after it.
   */
  markMeans: (goal: Goal, value: number) => {
    if (goal === 'retention') {
      return (
        `${share(value)} remembered is the target that leaves most of ` +
        'the material in the head.'
      )
    }
    if (goal === 'date') {
      return (
        `${many(value, 'day')} off is the first day this preset's own budget ` +
        'gets through the material.'
      )
    }
    return `${many(value, 'minute')} a day is the shortest day the clock no longer cuts short.`
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
  /**
   * What a goal of a date comes to, as one telling: what the day costs, what
   * the preset keeps against it, and how far that gets. Every clause that has
   * nothing to say is left out rather than said as a nought.
   */
  dated: (needed: number, standing: number, part: number, owed: number, met: boolean) => {
    const head =
      `Getting through it by then takes ${many(needed, 'minute')} a day, ` +
      `and this preset keeps ${count(standing)}`
    // Where no day of review gets there, the figure is a floor and not a cost.
    if (!met) {
      return (
        'No day of review gets through the material by that day: it would take more than ' +
        `${many(needed, 'minute')} a day, and this preset keeps ${count(standing)}.`
      )
    }
    if (part >= 1) return `${head}, which gets through all of it in time.`
    const owing = owed > 0 ? `, leaving ${many(owed, 'card')} owed on the day` : ''
    return `${head}, which gets ${count(part * 100)}% of it through by then${owing}.`
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
