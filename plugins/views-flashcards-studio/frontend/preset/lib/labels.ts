// Stryker disable StringLiteral: interface copy
import type { BudgetUnit, Goal, Rule } from '../types'
import type { Field } from './fields'

/** What each of the three goals is offered as: the value it steers. */
export const GOALS: Record<Goal, string> = {
  minutes: 'Minutes per day',
  retention: 'Retention',
  date: 'Target date',
}

/**
 * The label of each of the two rules for the learned state: a name, since a
 * control offers names and the row below it explains what they mean.
 */
export const RULES: Record<Rule, string> = {
  interval: 'By days',
  retention: 'By remembering',
}

/** What each of the two units a budget is spent in is offered as. */
export const BUDGET_UNITS: Record<BudgetUnit, string> = {
  cards: 'Cards',
  shows: 'Reviews',
}

/** The name of each setting, and a one-line description. */
export const FIELDS: Record<Field, readonly [string, string]> = {
  newADay: ['New per day', 'Maximum new cards per day.'],
  reviewsADay: ['Reviews per day', 'Maximum review cards per day.'],
  retention: ['Retention', 'Target share of cards you recall.'],
  minutesADay: ['Minutes per day', 'Daily review time, measured by answer times.'],
  byDate: ['Target date', 'Date by which all cards should be learned.'],
  counts: ['Budget unit', 'What the daily budget counts.'],
  learned: [
    'Learned when',
    'By days: reviews are at least the set number of days apart. ' +
      'By remembering: the recall chance today is at least the target.',
  ],
  interval: ['Days apart', 'Review interval at which a card counts as learned.'],
  backlog: [
    'Overdue share',
    'Percent of a session spent on overdue cards before new cards. ' +
      'At 0, new cards come first.',
  ],
  load: [
    'Load by weekday',
    'Percent of the daily budget for each weekday. ' +
      'Cards due on a 0% day stay overdue until the next day with load.',
  ],
  hasEvenLoad: [
    'Even load',
    'Move cards to lighter days within their interval. ' +
      'Cards with no other allowed day stay on their due day.',
  ],
}
