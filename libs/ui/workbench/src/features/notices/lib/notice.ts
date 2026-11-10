/**
 * What a notice is, as plain values. No DOM, no clock, no measurement.
 */

import type { TallyUnit, Severity } from './tally'

/**
 * How long a notice is displayed.
 *
 * `holds` is displayed while its source keeps re-submitting it. `read` is
 * removed once it has been up long enough to be read. `kept` stays until the
 * user dismisses it.
 */
export type Stay = 'holds' | 'read' | 'kept'

/**
 * One message of the window: a task running in the background, a standing
 * state, or an event that happened.
 *
 * `done` and `total` count the work in progress, not the size of what it is
 * applied to: one change counts as one, however large the changed object.
 */
export interface Notice {
  readonly id: string
  readonly text: string
  readonly about?: string
  readonly done?: number
  readonly total?: number
  readonly counting?: TallyUnit
  readonly isWorking?: boolean
  readonly severity?: Severity
  readonly stay?: Stay
  readonly isUserStarted?: boolean
}

/** One piece of work a window is doing behind itself, as it is answered for. */
export interface Task {
  readonly id: string
  readonly label: string
  readonly about: string
  readonly error: string
  readonly isUserStarted: boolean
  readonly done?: number
  readonly total?: number
  readonly counting?: TallyUnit
}

/**
 * One piece of work as a notice.
 *
 * What stopped the work is the card title: it is the sentence the user acts on,
 * and a card displays only its first words.
 */
export const getTaskNotice = (task: Task): Notice => {
  const error = task.error
  const completed = task.done
  const total = task.total

  return {
    id: task.id,
    text: error || task.label,
    about: task.about,
    isWorking: error === '',
    isUserStarted: task.isUserStarted || error !== '',
    ...(error ? { severity: 'error' as const, stay: 'kept' as const } : {}),
    // Stryker disable next-line ConditionalExpression,EqualityOperator: undefined > 0 is false, so total > 0 already rejects an undefined total.
    ...(completed !== undefined && total !== undefined && total > 0
      ? { done: completed, total: total, ...(task.counting ? { counting: task.counting } : {}) }
      : {}),
  }
}
