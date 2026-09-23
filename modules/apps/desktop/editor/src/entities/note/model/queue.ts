/**
 * Save timers, settling coordination, and read/write execution for open notes.
 */
import type { LinkAddress, NoteResult, WriteResult } from '../lib/note'
import type { ErrorCode } from '@/shared/errors'
import type { Notes } from '../lib/noteTypes'
import type { Event, NoteBaseline, NoteErrorCode } from '../lib/tab'

export const errorOf = (from: ErrorCode): NoteErrorCode => {
  if (from === 'deckTooLarge') return 'tooLarge'
  if (from === 'notAStencil' || from === 'notADeck' || from === 'notAPreset') return 'notANote'
  return from === 'missing' || from === 'occupied' || from === 'unnameable' ? 'unreadable' : from
}

export function createNoteQueue(
  core: Notes,
  turn: (id: string, event: Event) => void,
  hasTab: (id: string) => boolean,
  setAddress: (id: string, address: LinkAddress | null) => void,
  resumeClosing: (id: string) => void,
) {
  const timers = new Map<string, ReturnType<typeof setTimeout>>()
  const settling = new Map<string, () => void>()

  const arm = (id: string, after: number): void => {
    clearTimeout(timers.get(id))
    timers.set(
      id,
      setTimeout(() => {
        timers.delete(id)
        turn(id, { kind: 'fired' })
      }, after),
    )
  }

  const disarm = (id: string): void => {
    clearTimeout(timers.get(id))
    timers.delete(id)
  }

  const finishSettle = (id: string): void => {
    settling.get(id)?.()
    settling.delete(id)
  }

  const settle = (id: string): Promise<void> =>
    new Promise((done) => {
      if (!hasTab(id)) return done()
      settling.set(id, done)
      turn(id, { kind: 'settling' })
    })

  const read = async (id: string, path: string, generation: number): Promise<void> => {
    let answered: NoteResult
    try {
      answered = await core.read(path)
    } catch {
      // Core read failed.
      turn(id, { kind: 'read', generation, answer: { kind: 'error', error: 'unreachable' } })
      return
    }
    setAddress(id, answered.ok ? (answered.value.link ?? null) : null)
    turn(id, {
      kind: 'read',
      generation,
      answer: answered.ok
        ? { kind: 'body', body: answered.value.body, at: answered.value.at ?? '' }
        : answered.error === 'missing'
          ? { kind: 'missing' }
          : { kind: 'error', error: errorOf(answered.error) },
    })
  }

  const write = async (
    id: string,
    path: string,
    body: string,
    baseline: NoteBaseline | null,
  ): Promise<void> => {
    let answered: WriteResult
    try {
      answered = await core.write(path, body, baseline)
    } catch {
      // Core write failed.
      turn(id, { kind: 'written', answer: { kind: 'error', error: 'unreachable' } })
      return
    }
    turn(id, {
      kind: 'written',
      answer: answered.ok
        ? { kind: 'ok', at: answered.value.at ?? '' }
        : answered.error === 'changed'
          ? { kind: 'changed' }
          : { kind: 'error', error: errorOf(answered.error) },
    })
    resumeClosing(id)
  }

  const forget = (id: string): void => {
    disarm(id)
    finishSettle(id)
  }

  return {
    arm,
    disarm,
    finishSettle,
    settle,
    read,
    write,
    forget,
  }
}
