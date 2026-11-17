/** One setting of a preset that the user may set directly. */
export type Field =
  | 'minutesADay'
  | 'newADay'
  | 'reviewsADay'
  | 'retention'
  | 'learned'
  | 'interval'
  | 'byDate'
  | 'counts'
  | 'backlog'
  | 'load'
  | 'hasEvenLoad'

/** Every setting the receipt has a row for, in the order they stand in. */
export const FIELDS: readonly Field[] = [
  'newADay',
  'reviewsADay',
  'learned',
  'interval',
  'retention',
  'minutesADay',
  'byDate',
  'counts',
  'backlog',
  'load',
  'hasEvenLoad',
]
