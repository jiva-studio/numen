/** What the schema's values for a preset are called in the window's own words. */
import {
  BudgetName as BudgetNames,
  BudgetUnit as BudgetUnits,
  Rule as Rules,
  StopReason as StopReasons,
} from '@numen/protocol'
import { namesOf } from '@numen/wire'
import type { BudgetName, BudgetUnit, Rule, StopReason } from '../lib/presets'

/** What counts as learned. */
export const LEARNED: Record<Rules, Rule | null> = {
  [Rules.UNSPECIFIED]: null,
  [Rules.INTERVAL]: 'interval',
  [Rules.RETENTION]: 'retention',
}

export const RULING = namesOf<Rule, Rules>(LEARNED)

/** The unit a budget is spent in. */
export const COUNTED: Record<BudgetUnits, BudgetUnit | null> = {
  [BudgetUnits.UNSPECIFIED]: null,
  [BudgetUnits.CARDS]: 'cards',
  [BudgetUnits.SHOWS]: 'shows',
}

export const COUNTING = namesOf<BudgetUnit, BudgetUnits>(COUNTED)

/**
 * Why a preset schedules nothing. A preset that schedules names no reason, and
 * so does one the schema has nothing to say about.
 */
export const STOPPED: Record<StopReasons, StopReason> = {
  [StopReasons.UNSPECIFIED]: 'none',
  [StopReasons.NOTHING]: 'none',
  [StopReasons.NO_MINUTES]: 'noMinutes',
  [StopReasons.NO_CARDS]: 'noCards',
  [StopReasons.NO_DAY]: 'noDay',
  [StopReasons.PAST_DAY]: 'pastDay',
  [StopReasons.NO_LOAD]: 'noLoad',
  [StopReasons.NO_WEEK]: 'noWeek',
}

/**
 * The budget a day ran out of. A budget the schema has nothing to say about is
 * left out of what the day closed on.
 */
export const CLOSED: Record<BudgetNames, BudgetName | null> = {
  [BudgetNames.UNSPECIFIED]: null,
  [BudgetNames.MINUTES_A_DAY]: 'minutesADay',
  [BudgetNames.NEW_A_DAY]: 'newADay',
  [BudgetNames.REVIEWS_A_DAY]: 'reviewsADay',
  [BudgetNames.BY_DATE]: 'byDate',
  [BudgetNames.BACKLOG]: 'backlog',
  [BudgetNames.PAUSED]: 'paused',
}
