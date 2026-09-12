/**
 * What a notice is, as plain values. No DOM, no clock, no measurement.
 */

import type { TallyUnit, Tone } from './tally'

export type { Tone }

/**
 * How long a notice stands.
 *
 * `holds` is drawn while whoever hands it in keeps handing it in. `read` goes
 * once it has been up long enough to have been read. `kept` stands until a
 * person puts it away.
 */
export type Stay = 'holds' | 'read' | 'kept'

/**
 * One thing the window has to say: something running behind it, something that
 * is so, or something that happened.
 *
 * `done` and `total` are the work in hand, not the size of what it is being
 * done to: one change is one thing, however large the thing changed.
 */
export interface Notice {
  /** Whatever the caller addresses this notice by. Never read, only handed back. */
  readonly id: string
  /** What is happening, in the words it is to be shown by. */
  readonly says: string
  /** What it is happening to, or why it stopped, when that is worth saying. */
  readonly about?: string
  /** Where it has got to, when there is a total to count against. */
  readonly done?: number
  readonly total?: number
  /** What that count counts. */
  readonly counting?: TallyUnit
  /** Whether it is running now, or is a fact that is simply so. */
  readonly working?: boolean
  /** How it reads. Plain unless said otherwise. */
  readonly tone?: Tone
  /** How long it stands. Held unless said otherwise. */
  readonly stay?: Stay
  /**
   * Whether a person asked for this and is waiting to be told it began. One of
   * these is drawn the moment it arrives.
   */
  readonly asked?: boolean
}

/** One piece of work a window is doing behind itself, as it is answered for. */
export interface Task {
  readonly id: string
  /** The action or operation being performed (e.g. "Transcribing"). */
  readonly action?: string
  readonly doing: string
  /** The subject/target of the task (e.g. filename). */
  readonly target?: string
  readonly about: string
  /** Explanation if the task failed. */
  readonly failureReason?: string
  readonly failed: string
  /** Whether the task was initiated by user request. */
  readonly isUserRequested?: boolean
  readonly asked: boolean
  /** How far it has got, where there is a total to count against. */
  readonly completedCount?: number
  readonly done?: number
  readonly totalCount?: number
  readonly total?: number
  /** Unit of measurement for the count. */
  readonly unit?: TallyUnit
  readonly counting?: TallyUnit
}

/**
 * One piece of work as a notice.
 *
 * What stopped a piece of work is what its card is called: it is the sentence a
 * person acts on, and the room on a card is the words at the front of it.
 */
export const createNotice = (task: Task): Notice => {
  const failure = task.failureReason ?? task.failed
  const action = task.action ?? task.doing
  const subject = task.target ?? task.about
  const isUserReq = task.isUserRequested ?? task.asked
  const completed = task.completedCount ?? task.done
  const total = task.totalCount ?? task.total
  const unit = task.unit ?? task.counting

  return {
    id: task.id,
    says: failure || action,
    about: subject,
    working: failure === '',
    asked: isUserReq || failure !== '',
    ...(failure ? { tone: 'alarm' as const, stay: 'kept' as const } : {}),
    ...(completed !== undefined && total !== undefined && total > 0
      ? { done: completed, total: total, ...(unit ? { counting: unit } : {}) }
      : {}),
  }
}

/**
 * The notices worth drawing: the ones that have something to say.
 *
 * A notice with no words is one nobody could read.
 */
export const readable = (notices: readonly Notice[]): readonly Notice[] =>
  notices.filter((notice) => notice.says !== '')

/** What a notice counts against, for the ones that count anything. */
export const tallyOf = (notice: Notice): { done: number; total: number } | undefined =>
  notice.total === undefined || notice.done === undefined
    ? undefined
    : { done: notice.done, total: notice.total }

/**
 * What is still worth remembering as put away: the notices that are still
 * there. Work that ends and begins again is news, and says so.
 */
export const getStillAway = (
  away: ReadonlySet<string>,
  notices: readonly Notice[],
): ReadonlySet<string> => {
  const here = new Set(readable(notices).map((notice) => notice.id))
  return new Set([...away].filter((id) => here.has(id)))
}
