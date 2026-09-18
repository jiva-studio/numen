import { StopReason } from '@numen/protocol'
/** What a preset tab says: the one control, the settings under it, and what went wrong. */
import { many, percent, plural } from '@numen/ui'
import type { ErrorCode } from '@/shared/errors'
import type { BudgetUnit, Goal, Rule } from './types'
import type { Field } from './lib/fields'

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
  evenLoad: [
    'Even load',
    'Whether a card is leaned towards a lighter day of the days its own interval allows. ' +
      'A card whose interval allows no other day stays where it fell.',
  ],
}

/**
 * What a read of the preset encountered as an error. An error only a write answers
 * with, and one only a deck or a stencil encounters, has no sentence here
 * and is said in the one line under it.
 */
const READING: Record<ErrorCode, string | null> = {
  missing: 'This preset is no longer in the vault, so what stands here is what was last read.',
  notANote: 'What stands at this path is not a note, so there are no settings in it to read.',
  notText: 'This file is not text, so there are no settings in it to read.',
  tooLarge: 'This note is longer than the window reads, so none of its settings were read.',
  unreadable: 'The frontmatter of this note cannot be read, so none of its settings were read.',
  notAPreset: 'That note is not a preset, and the defaults stand.',
  bodyUnwritable: null,
  occupied: null,
  unnameable: null,
  notAStencil: null,
  notADeck: null,
  deckTooLarge: null,
  unreachable: 'The vault would not answer for this preset, and did not say why.',
}

/** The read encountered an error and the vault named no reason the window knows. */
const UNREAD = 'This preset could not be read, and the vault named no reason.'

/**
 * What a write of the settings was refused for. Each says where the settings
 * stand, which is in the tab: a failed write leaves the file as it was.
 */
const WRITING: Record<ErrorCode, string | null> = {
  missing:
    'This preset is no longer in the vault, so nothing was written. ' +
    'These settings are still here.',
  notANote:
    'What stands at this path is not a note, so nothing was written. ' +
    'These settings are still here.',
  notText: 'This file is not text, so nothing was written. These settings are still here.',
  tooLarge:
    'This note is longer than the window writes, so nothing was written. ' +
    'These settings are still here.',
  unreadable:
    'The frontmatter of this note cannot be read, so nothing was written. ' +
    'These settings are still here.',
  bodyUnwritable:
    'This note begins where its frontmatter should, so nothing was written. ' +
    'These settings are still here.',
  occupied:
    'A file stands where this note goes, so nothing was written. ' +
    'These settings are still here.',
  notAPreset: 'That note is not a preset, so nothing was written. These settings are still here.',
  unnameable: null,
  notAStencil: null,
  notADeck: null,
  deckTooLarge: null,
  unreachable: 'The vault would not answer for this preset, and did not say why.',
}

/** The write was refused and the vault named no reason the window knows. */
const UNWRITTEN =
  'These settings could not be written, and the vault named no reason. They are still here.'

/**
 * Why the preset schedules nothing, one sentence to each verdict the vault may
 * hand over. A verdict scheduling something says nothing. Every value stands
 * here, so a verdict added to the schema is one this window is made to answer.
 */
const STOPPED: Record<StopReason, string> = {
  [StopReason.UNSPECIFIED]: '',
  [StopReason.NOTHING]: '',
  [StopReason.NO_MINUTES]:
    'No minutes a day: this preset schedules nothing, and every deck pointing at it stops.',
  [StopReason.NO_CARDS]:
    'No cards a day: this preset schedules nothing, and every deck pointing at it stops.',
  [StopReason.NO_DAY]:
    'This preset aims at no day, so it schedules nothing. ' +
    'Name the day the material is to be in the head.',
  [StopReason.PAST_DAY]:
    'This preset is past the day it aimed at. Its budget is spent, and it schedules nothing.',
  [StopReason.NO_LOAD]:
    'Today carries none of this load, so this preset schedules nothing today. ' +
    'The next day that carries some picks its cards up.',
  [StopReason.NO_WEEK]:
    'No day of the week carries any of this load, so this preset schedules ' +
    'nothing on any of them, and every deck pointing at it stops.',
}

/** A count as a person reads one. */
const count = (value: number): string => `${Math.round(value)}`

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
      goal === 'minutes' ? `${many(at.value, 'minute')} a day` : `${percent(at.value)} remembered`
    const buys =
      goal === 'minutes'
        ? `${many(at.reviews, 'card')} a session`
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
    if (goal === 'retention') return percent(value)
    return goal === 'date' ? `${count(value)} d` : `${count(value)} min`
  },
  /**
   * What the control is acting on, said over the picture: the decks that point
   * here and the material they hold between them.
   */
  material: (decks: number, cards: number, overdue: number, fresh: number) => {
    const said = [
      { figure: count(decks), name: plural(decks, 'deck') },
      { figure: count(cards), name: plural(cards, 'card') },
    ]
    // A figure standing at nothing is left out.
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
          : {
              figure: count(learns),
              name: plural(learns, 'day to learn it', 'days to learn it'),
            }
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
  stopped: (why: StopReason) => STOPPED[why],
  /** What is wrong with the file, said above the control. */
  problems: 'What is wrong with this preset',
  /** The file moved under the window, and the two answers to that. */
  changed: 'This file changed on disk, so nothing was written.',
  reads: 'Read it again',
  /** What a read failed for. */
  notRead: (error: ErrorCode) => READING[error] ?? UNREAD,
  /** What a write was refused for, and where the settings stand after it. */
  notSaved: (error: ErrorCode) => WRITING[error] ?? UNWRITTEN,
  /** The vault answered a read with neither settings nor a reason. */
  unreachable: 'The vault would not answer for this preset, and did not say why.',
  /** The vault answered a write with neither a file nor a reason. */
  unwritten: 'The vault would not take these settings, and did not say why. They are still here.',
  /** The vault would not work the picture out. */
  noCurve: 'The vault would not work this picture out, and did not say why.',
}
