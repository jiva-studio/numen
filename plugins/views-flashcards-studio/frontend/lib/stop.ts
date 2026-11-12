/** StopReason is why a preset schedules no card for the day. */
export const StopReason = {
  UNSPECIFIED: '',
  NOTHING: 'nothing',
  NO_MINUTES: 'no_minutes',
  NO_CARDS: 'no_cards',
  NO_DAY: 'no_day',
  PAST_DAY: 'past_day',
  NO_LOAD: 'no_load',
  NO_WEEK: 'no_week',
} as const
export type StopReason = (typeof StopReason)[keyof typeof StopReason]

/** BudgetName is a limit that closes a day of review. */
export const BudgetName = {
  UNSPECIFIED: '',
  MINUTES_A_DAY: 'minutes_a_day',
  NEW_A_DAY: 'new_a_day',
  REVIEWS_A_DAY: 'reviews_a_day',
  BY_DATE: 'by_date',
  BACKLOG: 'backlog',
  PAUSED: 'paused',
} as const
export type BudgetName = (typeof BudgetName)[keyof typeof BudgetName]
