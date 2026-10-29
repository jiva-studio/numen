/**
 * The notices shown in the window's corner.
 *
 * Everything the window reports is one card: work in progress, a current state, and
 * the outcome of the last request. A new work item, state or outcome is an entry in one of
 * the three lists and requires no change here.
 */
import { createNotice } from '@numen/shell'
import type { Notice, Stay, Severity } from '@numen/shell'
import type { Task } from './task'
import { isTextSearchOnly, type IndexCoverage } from './coverage'
import type { WindowMessage } from './messages'

/** The sentences the corner draws that are the window's own. */
export interface Words {
  /** The vault is present and the window is not notified when it changes. */
  readonly unwatched: string
  /** The vault itself could not be read. */
  readonly unread: string
  /** The vault is still being read for the first time. */
  readonly indexing: string
  /** The vault could not be read, so there is nothing on screen. */
  readonly nothingRead: string
  /** One way of asking is missing and nothing is going to bring it. */
  readonly wordsOnly: string
}

/** What is so about the window, whatever it is doing. */
export interface State {
  /** The vault is present and the window is not notified when it changes. */
  readonly unwatched: string
  /** The vault itself could not be read. */
  readonly unread: string
  /** What the window lost touch with. */
  readonly lost: string
  /** The vault is still being read for the first time. */
  readonly isIndexing: boolean
  /** Whether the vault holds a note to show at all. */
  readonly hasNote: boolean
}

/** How long a message of each severity stays in the corner. */
const stayBySeverity: Record<Severity, Stay> = {
  error: 'kept',
  warning: 'kept',
  info: 'read',
}

/** One state of the window, where it is in that state. */
const createStateNotices = (
  id: string,
  text: string,
  how: { about?: string; severity?: Severity; isUserStarted?: boolean } = {},
): readonly Notice[] =>
  text === ''
    ? []
    : [
        {
          id,
          text,
          about: how.about ?? '',
          severity: how.severity ?? 'info',
          isWorking: false,
          isUserStarted: how.isUserStarted ?? true,
          stay: 'holds',
        },
      ]

/**
 * Whether one reason is the other with what a pass was doing put in front of
 * it. That is how a reason arrives from the core: `doing this: what went
 * wrong`.
 */
const isSameReason = (outer: string, inner: string): boolean =>
  outer === inner || outer.endsWith(`: ${inner}`)

/**
 * The work that gets a card of its own.
 *
 * One failure stops every dependent pass with the same message. The card is the one
 * that states the message and nothing more, and of two stating the same message, the
 * one reported first.
 */
const dropDuplicateErrors = (tasks: readonly Task[]): readonly Task[] =>
  tasks.filter((task, index) => {
    return !tasks.some(
      (other, was) =>
        other.error !== '' &&
        isSameReason(task.error, other.error) &&
        (other.error.length < task.error.length || was < index),
    )
  })

/**
 * The cards the corner draws.
 *
 * Work comes first, because only work has moving numbers, and an arriving state or
 * message must not shift what the user is reading. Within each of the three groups
 * the order is first-seen order.
 */
export interface CornerContext {
  state: State
  vault: IndexCoverage
  words: Words
}

export const getCornerNotices = (
  tasks: readonly Task[],
  messages: readonly WindowMessage[],
  context: CornerContext,
): readonly Notice[] => {
  const { state, vault, words } = context
  const isWorking: readonly Notice[] = dropDuplicateErrors(tasks).map((task) =>
    createNotice({ ...task, done: task.completedCount }),
  )

  const so: Notice[] = [
    ...createStateNotices('unwatched', state.unwatched && words.unwatched, {
      about: state.unwatched,
      severity: 'warning',
    }),
    ...createStateNotices('unread', state.unread && words.unread, {
      about: state.unread,
      severity: 'warning',
    }),
    ...createStateNotices('lost', state.lost, { severity: 'warning' }),
    ...createStateNotices('indexing', state.isIndexing ? words.indexing : ''),
    // One at a time: a vault still being indexed has not finished indexing nothing.
    ...createStateNotices(
      'nothingRead',
      !state.isIndexing && !state.hasNote && state.unread ? words.nothingRead : '',
    ),
    // Reported once, regardless of whether anything is running.
    ...createStateNotices('wordsOnly', isTextSearchOnly(vault) ? words.wordsOnly : '', {
      isUserStarted: false,
    }),
  ]

  const messageNotices: Notice[] = messages.map((one) => ({
    id: one.id,
    text: one.text,
    isWorking: false,
    isUserStarted: true,
    severity: one.severity,
    stay: stayBySeverity[one.severity],
  }))

  return [...isWorking, ...so, ...messageNotices]
}
