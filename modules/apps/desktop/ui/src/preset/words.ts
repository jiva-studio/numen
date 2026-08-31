/** What a preset tab says: the one control, the settings under it, and what went wrong. */
import type { Counts, Goal, Rule } from './core'
import type { Field } from './curve'

/** What each of the three goals is offered as: the value it steers. */
const GOALS: Record<Goal, string> = {
  minutes: 'Minutes a day',
  retention: 'Retention',
  date: 'A date',
}

/** What each of the two rules for the learned is offered as. */
const RULES: Record<Rule, string> = {
  interval: 'When reviews are far enough apart',
  retention: 'When you would remember it today',
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
  learned: [
    'Counts as learned',
    'What makes a card one you have learned: reviews far enough apart, or a good chance of ' +
      'remembering it today.',
  ],
  interval: [
    'Days between reviews',
    'How far apart reviews stand before a card counts as learned.',
  ],
  backlog: [
    'Overdue share',
    'What part of a sitting goes to the overdue pile before new material is offered, in per cent. ' +
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
  evenLoad: [
    'Even load',
    'Whether a card is leaned towards a lighter day of the days its own interval allows. ' +
      'A card whose interval allows no other day stays where it fell.',
  ],
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
  ruleName: (rule: Rule) => RULES[rule],
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
  /**
   * What the bubble over the knob says: the value being held, and under it what
   * that value buys. Each line is short enough to be taken at a glance.
   *
   * A goal of a date says nothing about the overdue, since every card is to be
   * got through by that day and the day the pile goes says nothing. Where
   * nothing is overdue there is no such line either.
   */
  buys: (
    goal: Goal,
    at: {
      value: number
      reviews: number
      minutes: number
      horizon: number
      /** Null is nothing overdue, and -1 a pile the days projected do not clear. */
      clears: number | null
      /** How many card faces no pace reaches by the day, out of how many there are. */
      short: number
      cards: number
    },
  ): readonly string[] => {
    if (goal === 'date') {
      const said = [`${many(at.value, 'day')} off`, `${many(at.minutes, 'minute')} a day`]
      // A day that leaves every card time enough has nothing to say about it.
      if (at.short > 0) said.push(`${count(at.short)} of ${count(at.cards)} cannot get there`)
      return said
    }
    const held =
      goal === 'minutes' ? `${many(at.value, 'minute')} a day` : `${share(at.value)} remembered`
    const buys =
      goal === 'minutes'
        ? `${many(at.reviews, 'card')} a sitting`
        : `${many(at.minutes, 'minute')} a day`
    if (at.clears === null) return [held, buys]
    const gone =
      at.clears < 0
        ? `not within ${many(at.horizon, 'day')}`
        : `overdue gone in ${many(at.clears, 'day')}`
    return [held, buys, gone]
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
   * When the material is learned, and how much of it stands learned now, read
   * off the run the picture is drawn from. Where there is no day the whole of
   * it stands learned on, nothing is said in its place: what is left is what
   * stands learned today. A pace whose days run out with one still to learn
   * has a day to speak of and says it has not come.
   */
  learning: (learns: number | undefined, learned: number, cards: number) => {
    const now = { figure: `${count(learned)} of ${count(cards)}`, name: 'learned today' }
    if (learns === undefined) return [now]
    const when =
      learns < 0
        ? { figure: 'not yet', name: 'in the days ahead' }
        : learns === 0
          ? { figure: 'today', name: 'all of it learned' }
          : { figure: count(learns), name: learns === 1 ? 'day to learn it' : 'days to learn it' }
    return [when, now]
  },
  fieldName: (field: Field) => FIELDS[field][0],
  fieldDetail: (field: Field) => FIELDS[field][1],
  /** A share in hundredths, read out beside the track it is moved along. */
  percent: (value: number) => `${count(value)}%`,
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
  /** The value the control stands at, in the units of its goal. */
  value: (goal: Goal, value: number, day: string) => {
    if (goal === 'retention') return `${share(value)} remembered`
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
