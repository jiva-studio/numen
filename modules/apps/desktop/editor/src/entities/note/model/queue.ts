/**
 * Save timers, flush coordination, and read/write execution for open notes.
 */
import type { LinkAddress, NoteResult, WriteResult as CoreWriteResult } from '../lib/note'
import type { ErrorCode } from '@/shared/errors'
import type { Notes } from '../lib/noteTypes'
import type { Event, NoteBaseline, NoteErrorCode, ReadResult, WriteResult } from '../lib/tab'

export const getNoteErrorCode = (from: ErrorCode): NoteErrorCode => {
  if (from === 'deckTooLarge') return 'tooLarge'
  if (from === 'notAStencil' || from === 'notADeck' || from === 'notAPreset') return 'notANote'
  return from === 'missing' || from === 'occupied' || from === 'unnameable' ? 'unreadable' : from
}

export function createNoteQueue(
  core: Notes,
  dispatch: (id: string, event: Event) => void,
  hasTab: (id: string) => boolean,
  setAddress: (id: string, address: LinkAddress | null) => void,
  resumeClosing: (id: string) => void,
) {
  const timers = new Map<string, ReturnType<typeof setTimeout>>()
  const flushWaiters = new Map<string, () => void>()

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
      // Core read failed.
      dispatch(id, { kind: 'read', generation, answer: { kind: 'error', error: 'unreachable' } })
      return
    }
    setAddress(id, answered.ok ? (answered.value.link ?? null) : null)
    let answer: ReadResult
    if (answered.ok) {
      answer = { kind: 'body', body: answered.value.body, at: answered.value.at ?? '' }
    } else if (answered.error === 'missing') {
      answer = { kind: 'missing' }
    } else {
      answer = { kind: 'error', error: getNoteErrorCode(answered.error) }
    }
    dispatch(id, { kind: 'read', generation, answer })
  }

  const write = async (
    id: string,
    path: string,
    body: string,
    baseline: NoteBaseline | null,
  ): Promise<void> => {
    let answered: CoreWriteResult
    try {
      answered = await core.write(path, body, baseline)
    } catch {
      // Core write failed.
      dispatch(id, { kind: 'written', answer: { kind: 'error', error: 'unreachable' } })
      return
    }
    let writeAnswer: WriteResult
    if (answered.ok) {
      writeAnswer = { kind: 'ok', at: answered.value.at ?? '' }
    } else if (answered.error === 'changed') {
      writeAnswer = { kind: 'changed' }
    } else {
      writeAnswer = { kind: 'error', error: getNoteErrorCode(answered.error) }
    }
    dispatch(id, { kind: 'written', answer: writeAnswer })
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
