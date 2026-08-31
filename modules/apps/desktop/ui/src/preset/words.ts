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

/** What each end of a goal's range is called, at the two bottom corners. */
const ENDS: Record<Goal, readonly [string, string]> = {
  minutes: ['a short day', 'an hour a day'],
  retention: ['easier to keep up', 'more of it back'],
  date: ['sooner', 'further off'],
}

/** What each setting is called, and the one line that says what it is. */
const FIELDS: Record<Field, readonly [string, string]> = {
  newADay: ['New a day', 'How many unseen cards a day holds.'],
  reviewsADay: ['Reviews a day', 'How many returning cards a day holds.'],
  retention: ['Retention', 'The share of cards recalled when they come round again.'],
  minutesADay: [
    'Minutes a day',
    'How long a day of review runs, spent against the time each answer took.',
  ],
  byDate: ['The day', 'The day the material is to be in the head.'],
  counts: ['Counts', "What a day's budget is spent on."],
  lightDays: ['Light days', 'The days of the week the load is cut on.'],
  evenLoad: ['Even load', 'Whether days are made to resemble each other.'],
}

/** A share of cards as a person reads one. */
const share = (value: number): string => value.toFixed(2)

/** A count as a person reads one. */
const count = (value: number): string => `${Math.round(value)}`

export const WORDS = {
  preset: 'Preset',
  /** What a preset tab is called before the vault has said what the note is. */
  newPreset: 'Preset',
  /** What the three segments are, said over them. */
  goal: 'Goal',
  goalName: (goal: Goal) => GOALS[goal],
  countsName: (counts: Counts) => COUNTS[counts],
  ends: (goal: Goal) => ENDS[goal],
  /** What the height of the picture is read in, said over it. */
  height: (goal: Goal) => (goal === 'minutes' ? 'Cards in a sitting' : 'Minutes a day'),
  /** One height of the picture, against the line it is the height of. */
  heightAt: (goal: Goal, value: number) =>
    goal === 'minutes' ? `${count(value)} cards` : `${count(value)} min`,
  /** One place along the picture, in the units of the goal's grid. */
  widthAt: (goal: Goal, value: number) => {
    if (goal === 'retention') return share(value)
    return goal === 'date' ? `${count(value)} d` : `${count(value)} min`
  },
  /** A day the card limits close before its minutes run out. */
  closed: (newADay: number, reviewsADay: number) =>
    `A longer day buys nothing here: ${count(newADay)} new and ${count(reviewsADay)} reviews a day ` +
    'close the day before its minutes run out.',
  fieldName: (field: Field) => FIELDS[field][0],
  fieldDetail: (field: Field) => FIELDS[field][1],
  settings: 'What the goal produced',
  /** The knob, and what it is announced as while it is moved. */
  knob: 'The goal of this preset',
  /** The two marks on the curve. */
  now: 'where this preset stands',
  suggested: 'suggested',
  /** The figure is the window's own arithmetic, and the answer is on its way. */
  about: 'about',
  aboutMeaning: 'a figure the window guessed while the application works out the honest one',
  /** The value the control stands at, in the units of its goal. */
  value: (goal: Goal, value: number, day: string) => {
    if (goal === 'retention') return `${share(value)} retention`
    if (goal === 'date') return `${count(value)} days to ${day}`
    return `${count(value)} minutes a day`
  },
  /** What standing there costs, in one sentence, in the words of the goal chosen. */
  costs: (goal: Goal, value: number, reviews: number, minutes: number, retained: number) => {
    if (goal === 'retention') {
      return `${share(value)} of it coming back is ${count(minutes)} minutes a day and ${count(reviews)} cards.`
    }
    if (goal === 'date') return `Being through it by then is ${count(minutes)} minutes a day.`
    return (
      `A sitting of ${count(value)} minutes puts ${count(reviews)} cards in front of you, ` +
      `and ${share(retained)} of the material comes back.`
    )
  },
  /** The arithmetic under a goal of a date, with the sum already done. */
  owing: (cards: number) => `${count(cards)} cards would still be owed on that day.`,
  needing: (needed: number, standing: number) =>
    `Getting through it by then is ${count(needed)} minutes a day, and this preset keeps ${count(standing)}.`,
  through: (part: number, standing: number) =>
    `At ${count(standing)} minutes a day, ${count(part * 100)}% of it is through by then.`,
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
