/**
 * Save timers, flush coordination, and read/write execution for open notes.
 */
import type { LinkAddress, Notes, NoteResult, WriteResult as CoreWriteResult } from './types'
import type { Event, NoteBaseline, NoteErrorCode, ReadResult, WriteResult } from './tab'

/** The codes a tab holds as its own. */
const TAB_CODES: ReadonlySet<string> = new Set<NoteErrorCode>([
  'tooLarge',
  'bodyUnwritable',
  'notANote',
  'notText',
  'unreadable',
  'unreachable',
])

/** The codes a vault answers with that a tab holds as another. */
const TAB_CODE_OF: Readonly<Record<string, NoteErrorCode>> = {
  deckTooLarge: 'tooLarge',
  notAStencil: 'notANote',
  notADeck: 'notANote',
  notAPreset: 'notANote',
  missing: 'unreadable',
  occupied: 'unreadable',
  unnameable: 'unreadable',
}

/** The code a tab holds for a code a vault answered with, and unreadable for one it has no word for. */
export const convertErrorCode = (from: string): NoteErrorCode =>
  TAB_CODE_OF[from] ?? (TAB_CODES.has(from) ? (from as NoteErrorCode) : 'unreadable')

export interface NoteQueueDeps {
  dispatch: (id: string, event: Event) => void
  hasTab: (id: string) => boolean
  setAddress: (id: string, address: LinkAddress | null) => void
  resumeClosing: (id: string) => void
}

export interface NoteWriteRequest {
  path: string
  body: string
  baseline: NoteBaseline | null
}

function toReadResult(answered: NoteResult): ReadResult {
  if (answered.ok) {
    return { kind: 'body', body: answered.value.body, at: answered.value.at ?? '' }
  }
  if (answered.error === 'missing') return { kind: 'missing' }
  return { kind: 'error', error: convertErrorCode(answered.error) }
}

function toWriteResult(answered: CoreWriteResult): WriteResult {
  if (answered.ok) return { kind: 'ok', at: answered.value.at ?? '' }
  if (answered.error === 'changed') return { kind: 'changed' }
  return { kind: 'error', error: convertErrorCode(answered.error) }
}

function createQueueTimers(dispatch: NoteQueueDeps['dispatch']) {
  const timers = new Map<string, ReturnType<typeof setTimeout>>()

  const startTimer = (id: string, after: number): void => {
    clearTimeout(timers.get(id))
    timers.set(
      id,
      setTimeout(() => {
        // Stryker disable next-line CallExpression: clearTimeout on a handle already fired does nothing, so a stale entry is never observed
        timers.delete(id)
        dispatch(id, { kind: 'timerFired' })
      }, after),
    )
  }

  const cancelTimer = (id: string): void => {
    clearTimeout(timers.get(id))
    // Stryker disable next-line CallExpression: clearTimeout on a handle already cleared does nothing, so a stale entry is never observed
    timers.delete(id)
  }

  return { startTimer, cancelTimer }
}

export function createNoteQueue(core: Notes, deps: NoteQueueDeps) {
  const { dispatch, hasTab, setAddress, resumeClosing } = deps
  const { startTimer, cancelTimer } = createQueueTimers(dispatch)
  const flushWaiters = new Map<string, () => void>()

  const resolveFlush = (id: string): void => {
    flushWaiters.get(id)?.()
    // Stryker disable next-line CallExpression: a stale waiter is a resolved promise callback, and calling it again changes nothing
    flushWaiters.delete(id)
  }

  const flushPending = (id: string): Promise<void> =>
    new Promise((done) => {
      if (!hasTab(id)) return done()
      flushWaiters.set(id, done)
      dispatch(id, { kind: 'flush' })
    })

  const read = async (id: string, path: string, generation: number): Promise<void> => {
    let answered: NoteResult
    try {
      answered = await core.read(path)
    } catch {
      dispatch(id, { kind: 'read', generation, answer: { kind: 'error', error: 'unreachable' } })
      return
    }
    setAddress(id, answered.ok ? (answered.value.link ?? null) : null)
    dispatch(id, { kind: 'read', generation, answer: toReadResult(answered) })
  }

  const write = async (id: string, request: NoteWriteRequest): Promise<void> => {
    let answered: CoreWriteResult
    try {
      answered = await core.write(request.path, request.body, request.baseline)
    } catch {
      dispatch(id, { kind: 'written', answer: { kind: 'error', error: 'unreachable' } })
      return
    }
    dispatch(id, { kind: 'written', answer: toWriteResult(answered) })
    resumeClosing(id)
  }

  const discard = (id: string): void => {
    cancelTimer(id)
    resolveFlush(id)
  }

  return {
    startTimer,
    cancelTimer,
    resolveFlush,
    flushPending,
    read,
    write,
    discard,
  }
}
