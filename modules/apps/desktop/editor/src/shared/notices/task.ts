/**
 * The work the application reports to the window while it goes on behind it.
 */
import type { TallyUnit } from '@numen/ui'

/**
 * One piece of work the application is doing behind the window.
 *
 * Every kind of work is one of these, and the window draws the list it is
 * given.
 */
export interface Task {
  /** What the work is called, so that the same work reported again replaces it. */
  readonly id: string
  /** The operation being performed (e.g. "Transcribing"). */
  readonly action?: string
  readonly doing: string
  /** The subject/target of the task (e.g. filename). */
  readonly target?: string
  readonly about: string
  /** How far it has got, where there is a total to count against. */
  readonly completedCount?: number
  readonly done: number
  readonly totalCount?: number
  readonly total: number
  /** What that count counts. */
  readonly unit?: TallyUnit
  readonly counting: TallyUnit
  /** Why it stopped, when it stopped with an error. */
  readonly failureReason?: string
  readonly failed: string
  /** Whether a person asked for this and is waiting to be told it began. */
  readonly isUserRequested?: boolean
  readonly asked: boolean
}
