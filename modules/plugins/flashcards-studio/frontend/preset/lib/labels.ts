// Stryker disable StringLiteral: interface copy
import type { BudgetUnit, Goal, Rule } from '../types'
import type { Field } from './fields'

/** What each of the three goals is offered as: the value it steers. */
export const GOALS: Record<Goal, string> = {
  minutes: 'Minutes a day',
  retention: 'Retention',
  date: 'A date',
}

/**
 * What each of the two rules for the learned is offered as: a name, since a
 * control offers names and the row under it says what they mean.
 */
export const RULES: Record<Rule, string> = {
  interval: 'By days',
  retention: 'By remembering',
}

/** What each of the two units a budget is spent in is offered as. */
export const BUDGET_UNITS: Record<BudgetUnit, string> = {
  cards: 'Cards',
  shows: 'Showings',
}

/** What each setting is called, and the one line that says what it is. */
export const FIELDS: Record<Field, readonly [string, string]> = {
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
