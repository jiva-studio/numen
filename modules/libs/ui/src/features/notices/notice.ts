/**
 * What a notice is, as plain values. No DOM, no clock, no measurement.
 */

import { rateOf, type TallyUnit, type Tone } from './activity/tally'

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
export const noticed = (task: Task): Notice => {
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

/** What one count stood at when it last moved, and when that was. */
export interface Movement {
  readonly done: number
  readonly rate: number
  readonly at: number
}

/**
 * How fast each count is moving, from where it last moved to where it is now.
 *
 * A reading that saw no movement leaves the mark where it is, so the stretch a
 * count stood still over is time the work took and is measured as such.
 *
 * A count that has gone is forgotten, and one that has just arrived is read once
 * before it has a rate.
 */
export const measured = (
  was: ReadonlyMap<string, Movement>,
  notices: readonly Notice[],
  at: number,
): ReadonlyMap<string, Movement> => {
  const moving = new Map<string, Movement>()
  for (const notice of readable(notices)) {
    const tally = tallyOf(notice)
    if (tally === undefined) continue
    const before = was.get(notice.id)
    if (before === undefined) {
      moving.set(notice.id, { done: tally.done, rate: 0, at })
      continue
    }
    if (tally.done === before.done) {
      moving.set(notice.id, before)
      continue
    }
    const rate = rateOf(before, tally.done, (at - before.at) / 1000)
    moving.set(notice.id, { done: tally.done, rate, at })
  }
  return moving
}

/**
 * How long work runs before it is worth a card, in milliseconds.
 *
 * Most passes are over before a person could read what they were called. A
 * notice somebody asked for does not wait: they are waiting for it.
 */
export const WAIT = 10_000

/** How long the shortest thing worth saying stands, in milliseconds. */
export const SETTLE = 4_000

/** How much longer it stands for each word it carries, in milliseconds. */
export const PER_WORD = 400

/** The length past which a notice is not read in passing. */
export const TOO_MUCH = 20

/**
 * How long a notice stands to be read, in milliseconds.
 *
 * Longer words are read for longer. Past twenty of them the time never runs
 * out: a list of names is something a person acts on, and it waits for them.
 */
export const dwellOf = (says: string, about = ''): number => {
  const words = `${says} ${about}`.split(/\s+/).filter((word) => word !== '')
  if (words.length > TOO_MUCH) return Infinity
  return SETTLE + words.length * PER_WORD
}

/**
 * When each notice standing now was first seen. One that has been here keeps
 * the moment it arrived; one that has gone is forgotten.
 */
export const arrivals = (
  was: ReadonlyMap<string, number>,
  notices: readonly Notice[],
  at: number,
): ReadonlyMap<string, number> =>
  new Map(readable(notices).map((notice) => [notice.id, was.get(notice.id) ?? at]))

/** Whether a notice has stood long enough to have been read. */
const over = (
  notice: Notice,
  arrived: ReadonlyMap<string, number>,
  at: number,
): boolean => at - (arrived.get(notice.id) ?? at) >= dwellOf(notice.says, notice.about)

/** The notices drawn: the ones that have lasted, less the ones put away. */
export const showing = (
  notices: readonly Notice[],
  arrived: ReadonlyMap<string, number>,
  away: ReadonlySet<string>,
  at: number,
  wait: number = WAIT,
): readonly Notice[] =>
  readable(notices).filter((notice) => {
    if (away.has(notice.id)) return false
    if (notice.stay === 'read') return !over(notice, arrived, at)
    return notice.asked || at - (arrived.get(notice.id) ?? at) >= wait
  })

/** The notices that have been read, and whose caller may forget them. */
export const finished = (
  notices: readonly Notice[],
  arrived: ReadonlyMap<string, number>,
  at: number,
): readonly string[] =>
  readable(notices)
    .filter((notice) => notice.stay === 'read' && over(notice, arrived, at))
    .map((notice) => notice.id)

/** How many cards stand at once. */
export const ROOM = 4

/**
 * The cards that stand, and how many are folded away behind them.
 *
 * What folds is what has been said and has gone right. Work, what is so, and
 * anything that stopped badly stand however many of them there are.
 */
export const folded = (
  drawn: readonly Notice[],
  room: number = ROOM,
): { shown: readonly Notice[]; over: number } => {
  if (drawn.length <= room) return { shown: drawn, over: 0 }
  const spare = drawn.filter(
    (notice) =>
      notice.stay !== undefined && notice.stay !== 'holds' && notice.tone !== 'alarm',
  )
  const away = new Set(spare.slice(0, drawn.length - room).map((notice) => notice.id))
  return { shown: drawn.filter((notice) => !away.has(notice.id)), over: away.size }
}

/**
 * What is still worth remembering as put away: the notices that are still
 * there. Work that ends and begins again is news, and says so.
 */
export const remembered = (
  away: ReadonlySet<string>,
  notices: readonly Notice[],
): ReadonlySet<string> => {
  const here = new Set(readable(notices).map((notice) => notice.id))
  return new Set([...away].filter((id) => here.has(id)))
}
