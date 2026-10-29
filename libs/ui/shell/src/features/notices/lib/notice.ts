/**
 * What a notice is, as plain values. No DOM, no clock, no measurement.
 */

import type { Tally, TallyUnit, Severity } from './tally'

export type { Severity }

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
  /** Whatever the caller addresses this notice by. Never read, only handed back. */
  readonly id: string
  /** What is happening, in the words it is to be shown by. */
  readonly text: string
  /** What it is happening to, or why it stopped, when that is worth saying. */
  readonly about?: string
  /** Where it has got to, when there is a total to count against. */
  readonly done?: number
  readonly total?: number
  /** What that count counts. */
  readonly counting?: TallyUnit
  /** Whether it is running now, or is a standing fact. */
  readonly isWorking?: boolean
  /** How severe it is: info unless set. */
  readonly severity?: Severity
  /** How long it is displayed. Defaults to held. */
  readonly stay?: Stay
  /**
   * Whether the user requested this and awaits confirmation that it began. One
   * of these is displayed as soon as it arrives.
   */
  readonly isUserStarted?: boolean
}

/** One piece of work a window is doing behind itself, as it is answered for. */
export interface Task {
  readonly id: string
  readonly label: string
  readonly about: string
  readonly error: string
  readonly isUserStarted: boolean
  /** How far it has got, where there is a total to count against. */
  readonly done?: number
  readonly total?: number
  /** What that count counts. */
  readonly counting?: TallyUnit
}

/**
 * One piece of work as a notice.
 *
 * What stopped the work is the card title: it is the sentence the user acts on,
 * and a card displays only its first words.
 */
export const createNotice = (task: Task): Notice => {
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
    ...(completed !== undefined && total !== undefined && total > 0
      ? { done: completed, total: total, ...(task.counting ? { counting: task.counting } : {}) }
      : {}),
  }
}

/**
 * The notices worth displaying: the ones with content.
 *
 * A notice with no text is unreadable.
 */
export const getReadable = (notices: readonly Notice[]): readonly Notice[] =>
  notices.filter((notice) => notice.text !== '')

/** What a notice counts against, for the ones that count anything. */
export const findTally = (notice: Notice): Tally | undefined =>
  notice.total === undefined || notice['done'] === undefined
    ? undefined
    : { completedCount: notice['done'], total: notice.total }

/**
 * The dismissed notices that remain relevant: those still active. Work that
 * ends and begins again is a new notice and is reported as one.
 */
export const getStillAway = (
  away: ReadonlySet<string>,
  notices: readonly Notice[],
): ReadonlySet<string> => {
  const here = new Set(getReadable(notices).map((notice) => notice.id))
  return new Set([...away].filter((id) => here.has(id)))
}
